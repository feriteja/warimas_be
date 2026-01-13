package order

import (
	"context"
	"errors"
	"fmt"
	"time"
	"warimas-be/internal/address"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/logger"
	"warimas-be/internal/payment"
	"warimas-be/internal/utils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Service interface {
	CreateFromSession(
		ctx context.Context,
		externalID string,
	) (*Order, error)
	OrderToPaymentProcess(ctx context.Context, sessionExternalID, externalID string, orderId uint) (*payment.PaymentResponse, error)
	GetOrders(ctx context.Context,
		filter *OrderFilterInput,
		sort *OrderSortInput,
		limit *int32,
		page *int32) ([]*Order, int64, error)
	GetOrderDetail(userID, orderID uint, isAdmin bool) (*Order, error)
	UpdateOrderStatus(orderID uint, status OrderStatus) error
	MarkAsPaid(ctx context.Context, referenceID, paymentRequestID, paymentProviderID string) error
	MarkAsFailed(ctx context.Context, referenceID, paymentRequestID, paymentProviderID string) error
	CreateSession(
		ctx context.Context,
		input model.CreateCheckoutSessionInput,
	) (*CheckoutSession, error)

	UpdateSessionAddress(
		ctx context.Context,
		externalID string,
		addressID string,
		guestID *string,
	) error
	ConfirmSession(
		ctx context.Context,
		sessionID string,
	) (*string, error)
	GetSession(
		ctx context.Context,
		externalID string,
	) (*CheckoutSession, error)
	GetPaymentOrderInfo(
		ctx context.Context,
		externalID string,
	) (*PaymentOrderInfoResponse, error)
}

type service struct {
	repo        Repository
	paymentRepo payment.Repository
	paymentGate payment.Gateway
	addressRepo address.Repository
}

func NewService(repo Repository, payRepo payment.Repository, payGate payment.Gateway, addressRepo address.Repository) Service {
	return &service{
		repo:        repo,
		paymentRepo: payRepo,
		paymentGate: payGate,
		addressRepo: addressRepo,
	}
}

func (s *service) CreateFromSession(
	ctx context.Context,
	externalID string,
) (*Order, error) {

	// 1. Load session
	session, err := s.repo.GetCheckoutSession(ctx, externalID)
	if err != nil {
		return nil, err
	}

	// 2. Validate session state
	if session.ConfirmedAt == nil {
		return nil, errors.New("checkout session not confirmed")
	}

	if session.Status != CheckoutSessionStatusPaid {
		return nil, errors.New("payment not completed")
	}

	// 3. IDEMPOTENCY CHECK
	existing, err := s.repo.GetOrderBySessionID(ctx, session.ID)
	if err == nil && existing != nil {
		return existing, nil
	}

	// 4. Create order domain
	order := &Order{
		UserID:      session.UserID,
		Status:      OrderStatus(model.OrderStatusPendingPayment),
		TotalAmount: uint(session.TotalPrice),
		Currency:    session.Currency,
		ExternalID:  utils.ExternalIDFromSession("pay", externalID),
	}

	// 5. Transaction boundary
	err = s.repo.CreateOrderTx(
		ctx,
		order,
		session,
	)
	if err != nil {
		return nil, err
	}

	return order, nil
}

// ✅ Create new order from cart
func (s *service) OrderToPaymentProcess(ctx context.Context, sessionExternalID string, externalID string, orderId uint) (*payment.PaymentResponse, error) {
	session, err := s.repo.GetCheckoutSession(context.Background(), sessionExternalID)
	if err != nil {
		return nil, err
	}

	userEmail := utils.GetUserEmailFromContext(ctx)

	var items []payment.XenditItem
	for _, s := range session.Items {
		items = append(items, payment.XenditItem{
			Name:     s.ProductName + " - " + s.VariantName,
			Quantity: s.Quantity,
			Price:    int64(s.Price),
		})
	}
	payResp, err := s.paymentGate.CreateInvoice(externalID,
		"userEmail",
		int64(session.TotalPrice),
		userEmail,
		items,
		payment.ChannelBCA)

	if err != nil {
		return nil, fmt.Errorf("failed to create payment invoice: %w", err)
	}

	p := &payment.Payment{
		OrderID:           orderId,
		ExternalReference: payResp.ProviderPaymentID,
		InvoiceURL:        payResp.InvoiceURL,
		Amount:            payResp.Amount,
		Status:            payResp.Status,
		PaymentMethod:     payResp.PaymentMethod,
		ChannelCode:       payResp.ChannelCode,
		PaymentCode:       payResp.PaymentCode,
		ExpireAt:          payResp.ExpirationTime,
	}

	err = s.paymentRepo.SavePayment(p)
	if err != nil {
		return nil, fmt.Errorf("failed to save payment: %w", err)
	}

	return payResp, nil
}

// ✅ Get list of orders (user or admin)

func (s *service) GetOrders(
	ctx context.Context,
	filter *OrderFilterInput,
	sort *OrderSortInput,
	limit *int32,
	page *int32,
) ([]*Order, int64, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "GetOrders"),
	)

	l := defaultLimit
	if limit != nil && *limit > 0 {
		l = *limit
	}
	if l > maxLimit {
		l = maxLimit
	}

	p := defaultPage
	if page != nil && *page > 0 {
		p = *page
	}

	offset := (p - 1) * l

	log.Info("fetching orders",
		zap.Int32("limit", l),
		zap.Int32("page", p),
		zap.Int32("offset", offset),
	)

	orders, err := s.repo.FetchOrders(ctx, filter, sort, l, offset)

	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.CountOrders(ctx, filter)

	ids := make([]uint, 0, len(orders))
	for _, o := range orders {
		ids = append(ids, o.ID)
	}

	itemsMap, err := s.repo.FetchOrderItems(ctx, ids)
	if err != nil {
		return nil, 0, err
	}

	for _, o := range orders {
		o.Items = itemsMap[o.ID]
	}

	log.Info("orders fetched",
		zap.Int("items_count", len(orders)),
		zap.Int64("total", total),
	)

	return orders, total, nil
}

// ✅ Get order detail (user only sees their own order)
func (s *service) GetOrderDetail(userID, orderID uint, isAdmin bool) (*Order, error) {
	order, err := s.repo.GetOrderDetail(orderID)
	if err != nil {
		return nil, err
	}

	if !isAdmin && order.UserID != &userID {
		return nil, fmt.Errorf("unauthorized: cannot access others' orders")
	}

	return order, nil
}

// ✅ Update order status (admin only)
func (s *service) UpdateOrderStatus(orderID uint, status OrderStatus) error {
	validStatuses := map[OrderStatus]bool{
		StatusPendingPayment: true,
		StatusPaid:           true,
		StatusFulFilling:     true,
		StatusCompleted:      true,
		StatusCanceled:       true,
	}

	if !validStatuses[status] {
		return fmt.Errorf("invalid status: %s", status)
	}

	return s.repo.UpdateOrderStatus(orderID, status)
}

func (s *service) MarkAsPaid(
	ctx context.Context,
	referenceID string,
	paymentRequestID string,
	paymentProviderID string,
) error {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "MarkAsPaid"),
		zap.String("reference_id", referenceID),
		zap.String("payment_request_id", paymentRequestID),
	)

	log.Info("mark as paid started")

	order, err := s.repo.GetByReferenceID(ctx, referenceID)
	if err != nil {
		log.Error("failed to fetch order", zap.Error(err))
		return err
	}

	// Idempotency guard
	if order.Status == "PAID" {
		log.Info("order already marked as PAID")
		return nil
	}

	// Optional safety check
	if order.Status == "FAILED" {
		log.Warn("cannot mark FAILED order as PAID")
		return fmt.Errorf("invalid status transition: FAILED -> PAID")
	}

	err = s.repo.UpdateStatusByReferenceID(
		ctx,
		referenceID,
		paymentRequestID,
		paymentProviderID,
		"PAID",
	)
	if err != nil {
		log.Error("failed to update order status to PAID", zap.Error(err))
		return err
	}

	log.Info("order successfully marked as PAID")
	return nil
}

func (s *service) MarkAsFailed(
	ctx context.Context,
	referenceID string,
	paymentRequestID string,
	paymentProviderID string,
) error {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "MarkAsFailed"),
		zap.String("reference_id", referenceID),
		zap.String("payment_request_id", paymentRequestID),
	)

	log.Info("mark as failed started")

	order, err := s.repo.GetByReferenceID(ctx, referenceID)
	if err != nil {
		log.Error("failed to fetch order", zap.Error(err))
		return err
	}

	// Idempotency guard
	if order.Status == "FAILED" {
		log.Info("order already marked as FAILED")
		return nil
	}

	// Optional safety check
	if order.Status == "PAID" {
		log.Warn("cannot mark PAID order as FAILED")
		return fmt.Errorf("invalid status transition: PAID -> FAILED")
	}

	err = s.repo.UpdateStatusByReferenceID(
		ctx,
		referenceID,
		paymentRequestID,
		paymentProviderID,
		"FAILED",
	)
	if err != nil {
		log.Error("failed to update order status to FAILED", zap.Error(err))
		return err
	}

	log.Info("order successfully marked as FAILED")
	return nil
}

func (s *service) CreateSession(
	ctx context.Context,
	input model.CreateCheckoutSessionInput,
) (*CheckoutSession, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "CreateSession"),
		zap.Int("item_count", len(input.Items)),
	)

	log.Info("create checkout session started")

	if len(input.Items) == 0 {
		log.Warn("checkout session creation with empty items")
		return nil, errors.New("checkout session must contain at least one item")
	}

	// 1. Resolve user (optional)
	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		log.Debug("creating checkout session as guest")
	}

	// 2. Validate variants & calculate price
	items := make([]CheckoutSessionItem, 0, len(input.Items))
	var subtotal int64 = 0

	for i, item := range input.Items {
		logItem := log.With(
			zap.Int("index", i),
			zap.String("variant_id", item.VariantID),
			zap.Int32("quantity", item.Quantity),
		)

		if item.Quantity <= 0 {
			logItem.Warn("invalid quantity")
			return nil, errors.New("quantity must be greater than zero")
		}

		variant, product, err := s.repo.GetVariantForCheckout(ctx, item.VariantID)
		if err != nil {
			logItem.Error("failed to get variant for checkout", zap.Error(err))
			return nil, err
		}

		if variant.Stock < item.Quantity {
			logItem.Warn(
				"insufficient stock",
				zap.Int32("stock", variant.Stock),
			)
			return nil, errors.New("product stock is not enough")
		}

		itemSubtotal := int64(variant.Price) * int64(item.Quantity)
		subtotal += itemSubtotal

		logItem.Debug(
			"item calculated",
			zap.String("variant_name", variant.Name),
			zap.String("product_name", product.Name),
			zap.Int64("price", int64(variant.Price)),
			zap.Int64("item_subtotal", itemSubtotal),
		)

		items = append(items, CheckoutSessionItem{
			ID:           uuid.New(),
			VariantID:    variant.ID,
			VariantName:  variant.Name,
			ProductName:  product.Name,
			Quantity:     int(item.Quantity),
			QuantityType: variant.QuantityType,
			ImageURL:     &variant.ImageURL, // already *string
			Price:        int(variant.Price),
			Subtotal:     int(itemSubtotal),
		})
	}

	// 3. Calculate fees
	tax := subtotal * 10 / 100
	var shippingFee int64 = 0
	var discount int64 = 0
	totalPrice := subtotal + tax + shippingFee - discount

	log.Info(
		"price calculated",
		zap.Int64("subtotal", subtotal),
		zap.Int64("tax", tax),
		zap.Int64("shipping_fee", shippingFee),
		zap.Int64("discount", discount),
		zap.Int64("total_price", totalPrice),
	)

	// 4. Create session model
	sessionID := uuid.New()
	session := &CheckoutSession{
		ID:          sessionID,
		ExternalID:  utils.ExternalIDFromSession("ck", sessionID.String()),
		Status:      CheckoutSessionStatusPending,
		Subtotal:    int(subtotal),
		Tax:         int(tax),
		ShippingFee: int(shippingFee),
		Discount:    int(discount),
		TotalPrice:  int(totalPrice),
		ExpiresAt:   time.Now().Add(30 * time.Minute),
	}

	if userID != 0 {
		session.UserID = &userID
	}

	log = log.With(
		zap.String("session_id", session.ExternalID),
		zap.String("status", string(session.Status)),
	)

	// 5. Persist in transaction
	if err := s.repo.CreateCheckoutSession(ctx, session, items); err != nil {
		log.Error("failed to create checkout session", zap.Error(err))
		return nil, err
	}

	log.Info("checkout session created successfully")

	return session, nil
}

func (s *service) UpdateSessionAddress(
	ctx context.Context,
	externalID string,
	addressID string,
	guestID *string,
) error {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "UpdateSessionAddress"),
		zap.String("session_id", externalID),
		zap.String("address_id", addressID),
	)

	// 1. Load checkout session
	session, err := s.repo.GetCheckoutSession(ctx, externalID)
	if err != nil {
		log.Error("failed to fetch checkout session", zap.Error(err))
		return err
	}

	// 2. Authorization
	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		log.Warn("failed to get user id from context", zap.Error(err))
	}

	if guestID != nil {
		guestUUID, err := uuid.Parse(*guestID)
		if err != nil {
			log.Warn("invalid guest id format", zap.String("guest_id", *guestID))
			return errors.New("invalid guest id")
		}

		if session.GuestID == nil || *session.GuestID != guestUUID {
			log.Warn("guest id mismatch")
			return errors.New("forbidden: guest ID mismatch")
		}

		log.Debug("authorized as guest", zap.String("guest_id", guestUUID.String()))

	} else {
		if session.UserID == nil || *session.UserID != userID {
			log.Warn("user does not own checkout session", zap.Uint("user_id", userID))
			return errors.New("forbidden: cannot update others' sessions")
		}

		log.Debug("authorized as user", zap.Uint("user_id", userID))
	}

	// 3. Session state validation
	if session.Status != CheckoutSessionStatusPending {
		log.Warn("checkout session is not editable",
			zap.String("status", string(session.Status)),
		)
		return errors.New("checkout session is not editable")
	}

	if time.Now().After(session.ExpiresAt) {
		log.Warn("checkout session expired",
			zap.Time("expires_at", session.ExpiresAt),
		)
		return errors.New("checkout session expired")
	}

	// 4. Fetch user address
	address, err := s.repo.GetUserAddress(ctx, addressID, userID)
	if err != nil {
		log.Error("failed to fetch user address", zap.Error(err))
		return err
	}

	// 5. Recalculate pricing
	shippingFee := s.calculateShippingFee(address, session.Items)
	tax := s.calculateTax(address, session.Subtotal)

	session.AddressID = &address.ID
	session.ShippingFee = shippingFee
	session.Tax = tax
	session.TotalPrice = session.Subtotal + tax + shippingFee - session.Discount

	log.Debug("pricing recalculated",
		zap.Int("shipping_fee", shippingFee),
		zap.Int("tax", tax),
		zap.Int("total_price", session.TotalPrice),
	)

	// 6. Persist changes
	if err := s.repo.UpdateSessionAddressAndPricing(ctx, session); err != nil {
		log.Error("failed to update session address and pricing", zap.Error(err))
		return err
	}

	log.Info("session address updated successfully")

	return nil
}

func (s *service) calculateShippingFee(
	address *address.Address,
	items []CheckoutSessionItem,
) int {
	// stub logic
	if address.City == "Jakarta" {
		return 10000
	}
	return 20000
}

func (s *service) calculateTax(
	address *address.Address,
	subtotal int,
) int {
	return subtotal * 10 / 100
}

func (s *service) ConfirmSession(
	ctx context.Context,
	externalID string,
) (*string, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "ConfirmSession"),
		zap.String("externa_idD", externalID),
	)

	userID, _ := utils.GetUserIDFromContext(ctx)

	log.Info("confirm checkout session started")

	// 1. Load session (with items)
	session, err := s.repo.GetCheckoutSession(ctx, externalID)
	if err != nil {
		log.Error("failed to load checkout session", zap.Error(err))
		return nil, err
	}

	log.Debug("checkout session loaded",
		zap.String("status", string(session.Status)),
		zap.Int("items_count", len(session.Items)),
	)

	// 2. Ownership check (if not guest)
	if session.UserID != nil && *session.UserID != userID {
		log.Warn("ownership check failed",
			zap.Uint("session_user_id", *session.UserID),
			zap.Uint("request_user_id", userID),
		)
		return nil, errors.New("forbidden")
	}

	// 3. Validate state
	if session.Status != CheckoutSessionStatusPending {
		log.Warn("invalid session status",
			zap.String("status", string(session.Status)),
		)
		return nil, errors.New("checkout session already confirmed")
	}

	if time.Now().After(session.ExpiresAt) {
		log.Warn("checkout session expired",
			zap.Time("expires_at", session.ExpiresAt),
		)
		return nil, errors.New("checkout session expired")
	}

	if session.AddressID == nil {
		log.Warn("shipping address not set")
		return nil, errors.New("shipping address not set")
	}

	if len(session.Items) == 0 {
		log.Warn("checkout session has no items")
		return nil, errors.New("checkout session has no items")
	}

	// 4. Re-validate stock & price
	for _, item := range session.Items {
		ok, err := s.repo.ValidateVariantStock(
			ctx,
			item.VariantID,
			item.Quantity,
		)
		if err != nil {
			log.Error("failed to validate variant stock",
				zap.String("variant_id", item.VariantID),
				zap.Int("quantity", item.Quantity),
				zap.Error(err),
			)
			return nil, err
		}
		if !ok {
			log.Warn("product out of stock",
				zap.String("variant_id", item.VariantID),
				zap.Int("quantity", item.Quantity),
			)
			return nil, errors.New("product out of stock")
		}
	}

	log.Info("stock validation passed")

	externalOrderID := utils.ExternalIDFromSession("pay", session.ID.String())

	order := &Order{
		UserID:      session.UserID,
		Status:      OrderStatus(model.OrderStatusPendingPayment),
		TotalAmount: uint(session.TotalPrice),
		Currency:    session.Currency,
		ExternalID:  externalOrderID,
	}

	err = s.repo.CreateOrderTx(
		ctx,
		order,
		session,
	)
	if err != nil {
		return nil, err
	}

	// 7. Persist changes
	err = s.repo.ConfirmCheckoutSession(ctx, session)
	if err != nil {
		log.Error("failed to confirm checkout session", zap.Error(err))
		return nil, err
	}

	_, err = s.OrderToPaymentProcess(ctx, session.ExternalID, externalOrderID, order.ID)

	log.Info("checkout session confirmed successfully",
		zap.String("final_status", string(session.Status)),
	)

	return &externalOrderID, nil
}

func (s *service) GetSession(
	ctx context.Context,
	externalID string,
) (*CheckoutSession, error) {

	userID, ok := utils.GetUserIDFromContext(ctx)
	session, err := s.repo.GetCheckoutSession(ctx, externalID)
	if err != nil {
		return nil, err
	}

	// Ownership check (if session is tied to a user)
	if session.UserID != nil && ok {
		if *session.UserID != userID {
			return nil, errors.New("forbidden")
		}
	}

	// Expiration handling (soft)
	if time.Now().After(session.ExpiresAt) &&
		session.Status == CheckoutSessionStatusPending {

		// Optional: mark expired lazily
		_ = s.repo.MarkSessionExpired(ctx, session.ID)
		session.Status = CheckoutSessionStatusExpired
	}

	return session, nil
}

func (s *service) GetPaymentOrderInfo(
	ctx context.Context,
	externalID string,
) (*PaymentOrderInfoResponse, error) {
	userID, ok := utils.GetUserIDFromContext(ctx)
	order, err := s.repo.GetOrderByExternalID(ctx, externalID)
	if err != nil {
		return nil, err
	}

	// Ownership check (if order is tied to a user)
	if order.UserID != nil && ok {
		if *order.UserID != userID {
			return nil, errors.New("forbidden")
		}
	}

	paymentData, err := s.paymentRepo.GetPaymentByOrder(order.ID)
	if err != nil {
		return nil, err
	}

	address, err := s.addressRepo.GetByID(ctx, order.AddressID)
	if err != nil {
		return nil, err
	}

	instructions := payment.GetInstructions(paymentData.PaymentMethod)

	instructions = payment.InjectVariables(
		instructions,
		payment.InstructionVars{
			"amount":       utils.FormatIDR(int64(order.TotalAmount)),
			"payment_code": paymentData.PaymentCode,
		},
	)

	paymentInfo := &PaymentOrderInfoResponse{
		OrderExternalID: externalID,
		Status:          PaymentStatus(paymentData.Status),
		TotalAmount:     int(order.TotalAmount),
		Currency:        order.Currency,
		ExpiresAt:       paymentData.ExpireAt,
		ShippingAddress: ShippingAddress{
			Name:         address.Name,
			ReceiverName: address.ReceiverName,
			Phone:        address.Phone,
			Address1:     address.Address1,
			Address2:     address.Address2,
			City:         address.City,
			Province:     address.Province,
			PostalCode:   address.Postal,
		},
		Payment: PaymentDetail{
			Method:       paymentData.PaymentMethod,
			PaymentCode:  &paymentData.PaymentCode,
			ReferenceID:  paymentData.ExternalReference,
			Instructions: instructions,
		},
	}

	return paymentInfo, nil

}
