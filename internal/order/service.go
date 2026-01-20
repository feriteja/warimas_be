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
	"warimas-be/internal/user"
	"warimas-be/internal/utils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Service interface {
	CreateFromSession(
		ctx context.Context,
		externalID string,
	) (*Order, error)
	OrderToPaymentProcess(ctx context.Context, session *CheckoutSession, externalID string, orderId uint) (*payment.PaymentResponse, error)
	GetOrders(
		ctx context.Context,
		filter *OrderFilterInput,
		sort *OrderSortInput,
		limit int32,
		page int32,
	) ([]*Order, int64, map[uuid.UUID][]address.Address, error)
	GetOrderDetail(ctx context.Context, orderID uint) (*Order, *address.Address, error)
	GetOrderDetailByExternalID(ctx context.Context, externalId string) (*Order, *address.Address, error)
	UpdateOrderStatus(ctx context.Context, orderID uint, status OrderStatus) error
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
	UpdateSessionPaymentMethod(
		ctx context.Context,
		externalID string,
		paymentMethod payment.ChannelCode,
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
	GetOrderForWebhook(
		ctx context.Context,
		externalID string,
	) (*Order, error)
}

type UserGateway interface {
	GetProfile(ctx context.Context, userID uint) (*user.Profile, error)
}

type service struct {
	repo        Repository
	paymentRepo payment.Repository
	paymentGate payment.Gateway
	addressRepo address.Repository
	userRepo    UserGateway
}

func NewService(repo Repository, payRepo payment.Repository, payGate payment.Gateway, addressRepo address.Repository, userRepo UserGateway) Service {
	return &service{
		repo:        repo,
		paymentRepo: payRepo,
		paymentGate: payGate,
		addressRepo: addressRepo,
		userRepo:    userRepo,
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
func (s *service) OrderToPaymentProcess(ctx context.Context, session *CheckoutSession, externalID string, orderId uint) (*payment.PaymentResponse, error) {
	userEmail := utils.GetUserEmailFromContext(ctx)

	var items []payment.XenditItem
	for _, s := range session.Items {
		items = append(items, payment.XenditItem{
			Name:     fmt.Sprintf("%s - %s", s.ProductName, s.VariantName),
			Quantity: s.Quantity,
			Price:    int64(s.Price),
		})
	}

	var userName string
	if session.UserID != nil && *session.UserID > 0 {
		userProfile, err := s.userRepo.GetProfile(ctx, uint(*session.UserID))
		if err == nil && userProfile != nil {
			if userProfile.FullName != nil {
				userName = *userProfile.FullName
			}
		} else {
			logger.FromCtx(ctx).Warn("failed to fetch user profile for invoice", zap.Error(err))
		}
	}

	// Fallback to address receiver name if profile name is missing
	if userName == "" && session.AddressID != nil {
		addr, err := s.addressRepo.GetByID(ctx, *session.AddressID)
		if err == nil && addr != nil {
			userName = addr.ReceiverName
		}
	}

	if userName == "" {
		userName = "Guest"
	}

	paymentMethod := payment.ChannelCode(payment.MethodGOPAY)
	if session.PaymentMethod != nil {
		paymentMethod = payment.ChannelCode(*session.PaymentMethod)

	}

	payResp, err := s.paymentGate.CreateInvoice(externalID,
		userName,
		int64(session.TotalPrice),
		userEmail,
		items,
		paymentMethod)

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

	err = s.paymentRepo.SavePayment(ctx, p)
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
	limit int32,
	page int32,
) ([]*Order, int64, map[uuid.UUID][]address.Address, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "GetOrders"),
	)

	// Defaults
	l := limit
	if l <= 0 {
		l = defaultLimit
	}
	if l > maxLimit {
		l = maxLimit
	}

	p := page
	if p <= 0 {
		p = defaultPage
	}

	offset := (p - 1) * l

	log.Info("fetching orders",
		zap.Int32("limit", l),
		zap.Int32("page", p),
		zap.Int32("offset", offset),
	)

	// Fetch orders
	orders, err := s.repo.FetchOrders(ctx, filter, sort, l, offset)
	if err != nil {
		log.Error("failed to fetch orders", zap.Error(err))
		return nil, 0, nil, err
	}

	// Count total
	total, err := s.repo.CountOrders(ctx, filter)
	if err != nil {
		log.Error("failed to count orders", zap.Error(err))
		return nil, 0, nil, err
	}

	if len(orders) == 0 {
		return orders, total, map[uuid.UUID][]address.Address{}, nil
	}

	// Collect order IDs & address IDs
	orderIDs := make([]int32, 0, len(orders))
	addressIDs := make([]uuid.UUID, 0, len(orders))

	for _, o := range orders {
		orderIDs = append(orderIDs, o.ID)
		addressIDs = append(addressIDs, o.AddressID)
	}

	// Fetch addresses in batch (IMPORTANT)
	addresses, err := s.addressRepo.GetByIDs(ctx, addressIDs)
	if err != nil {
		log.Error("failed to fetch addresses", zap.Error(err))
		return nil, 0, nil, err
	}

	// Map addressID -> []address.Address
	addressMap := make(map[uuid.UUID][]address.Address, len(addresses))
	for _, addr := range addresses {
		addressMap[addr.ID] = append(addressMap[addr.ID], addr)
	}

	// Fetch order items
	itemsMap, err := s.repo.FetchOrderItems(ctx, orderIDs)
	if err != nil {
		log.Error("failed to fetch order items", zap.Error(err))
		return nil, 0, nil, err
	}

	// Attach items
	for _, o := range orders {
		o.Items = itemsMap[o.ID]
	}

	log.Info("orders fetched",
		zap.Int("orders_count", len(orders)),
		zap.Int64("total", total),
	)

	return orders, total, addressMap, nil
}

// ✅ Get order detail (user only sees their own order), but admin could see everything
// GetOrderDetail returns order detail
// - User can only access their own order
// - Admin can access any order
func (s *service) GetOrderDetail(
	ctx context.Context,
	orderID uint,
) (*Order, *address.Address, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "GetOrderDetail"),
		zap.Uint("order_id", orderID),
	)

	log.Info("fetching order detail")

	// Fetch order
	order, err := s.repo.GetOrderDetail(ctx, orderID)
	if err != nil {
		log.Error("failed to fetch order detail", zap.Error(err))
		return nil, nil, err
	}

	if order == nil {
		log.Warn("order not found")
		return nil, nil, ErrOrderNotFound
	}

	// Get auth info
	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		log.Error("failed to get user id from context: unauthenticated")
		return nil, nil, ErrUnauthorized
	}

	userRole := utils.GetUserRoleFromContext(ctx)
	isAdmin := userRole == "ADMIN"

	// Authorization check
	if !isAdmin {
		if order.UserID == nil {
			log.Error("order user_id is nil")
			return nil, nil, fmt.Errorf("invalid order data")
		}

		if int32(userID) != *order.UserID {
			log.Warn("unauthorized order access attempt",
				zap.Uint("request_user_id", userID),
				zap.Int32("order_user_id", *order.UserID),
				zap.String("user_role", userRole),
			)
			return nil, nil, ErrUnauthorized
		}
	}

	// Fetch address
	addr, err := s.addressRepo.GetByID(ctx, order.AddressID)
	if err != nil {
		log.Error("failed to fetch address",
			zap.String("address_id", order.AddressID.String()),
			zap.Error(err),
		)
		return nil, nil, err
	}

	log.Info("order detail fetched successfully")

	return order, addr, nil
}

func (s *service) GetOrderDetailByExternalID(
	ctx context.Context,
	externalID string,
) (*Order, *address.Address, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "GetOrderDetail"),
		zap.String("external_id", externalID),
	)

	log.Info("fetching order detail")

	// Fetch order
	order, err := s.repo.GetOrderDetailByExternalID(ctx, externalID)
	if err != nil {
		log.Error("failed to fetch order detail", zap.Error(err))
		return nil, nil, err
	}

	if order == nil {
		log.Warn("order not found")
		return nil, nil, ErrOrderNotFound
	}

	// Get auth info
	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		log.Error("failed to get user id from context: unauthenticated")
		return nil, nil, ErrUnauthorized
	}

	userRole := utils.GetUserRoleFromContext(ctx)
	isAdmin := userRole == "ADMIN"

	// Authorization check
	if !isAdmin {
		if order.UserID == nil {
			log.Error("order user_id is nil")
			return nil, nil, fmt.Errorf("invalid order data")
		}

		if int32(userID) != *order.UserID {
			log.Warn("unauthorized order access attempt",
				zap.Uint("request_user_id", userID),
				zap.Int32("order_user_id", *order.UserID),
				zap.String("user_role", userRole),
			)
			return nil, nil, ErrUnauthorized
		}
	}

	// Fetch address
	addr, err := s.addressRepo.GetByID(ctx, order.AddressID)
	if err != nil {
		log.Error("failed to fetch address",
			zap.String("address_id", order.AddressID.String()),
			zap.Error(err),
		)
		return nil, nil, err
	}

	log.Info("order detail fetched successfully")

	return order, addr, nil
}

// ✅ Update order status (admin only)
func (s *service) UpdateOrderStatus(ctx context.Context, orderID uint, status OrderStatus) error {
	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "UpdateOrderStatus"),
		zap.Uint("order_id", orderID),
		zap.String("new_status", string(status)),
	)

	log.Info("update order status started")

	// 1. Fetch current order
	order, err := s.repo.GetOrderDetail(ctx, orderID)
	if err != nil {
		log.Error("failed to fetch order detail", zap.Error(err))
		return err
	}
	if order == nil {
		log.Warn("order not found")
		return ErrOrderNotFound
	}

	current := order.Status
	log = log.With(zap.String("current_status", string(current)))

	// Rule 4 & Terminal check: Cannot change status if already completed, cancelled or failed
	if current == OrderStatusCompleted || current == OrderStatusCancelled || current == OrderStatusFailed {
		log.Warn("cannot update order with terminal status")
		return fmt.Errorf("cannot update order with terminal status: %s", current)
	}

	// Rule 6: FAILED is free (can transition TO failed from any non-terminal state)
	if status == OrderStatusFailed {
		log.Info("transitioning to FAILED status")
		if err := s.repo.UpdateOrderStatus(ctx, orderID, status, nil); err != nil {
			log.Error("failed to update order status to FAILED", zap.Error(err))
			return err
		}
		log.Info("order status updated to FAILED successfully")
		return nil
	}

	// Allowed transitions map (Rule 1, 5)
	allowed := map[OrderStatus]map[OrderStatus]bool{
		OrderStatusPendingPayment: {
			OrderStatusPaid:      true,
			OrderStatusCancelled: true,
		},
		OrderStatusPaid: {
			OrderStatusAccepted:  true,
			OrderStatusCancelled: true,
		},
		OrderStatusAccepted: {
			OrderStatusShipped:   true,
			OrderStatusCancelled: true,
		},
		OrderStatusShipped: {
			OrderStatusCompleted: true,
		},
	}

	if targets, ok := allowed[current]; !ok || !targets[status] {
		log.Warn("invalid status transition")
		return fmt.Errorf("invalid status transition from %s to %s", current, status)
	}

	var invoiceNumber *string
	if status == OrderStatusAccepted {
		inv := utils.GenerateInvoiceNumber()
		invoiceNumber = &inv
		log.Info("generated invoice number", zap.String("invoice_number", inv))
	}

	if err := s.repo.UpdateOrderStatus(ctx, orderID, status, invoiceNumber); err != nil {
		log.Error("failed to update order status", zap.Error(err))
		return err
	}

	log.Info("order status updated successfully")
	return nil
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

	userId, _ := utils.GetUserIDFromContext(ctx)

	// 1. Validate variants & calculate price
	items := make([]CheckoutSessionItem, 0, len(input.Items))
	subtotal := 0

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
			logItem.Error(
				"failed to get variant for checkout",
				zap.Error(err),
			)
			return nil, err
		}

		itemSubtotal := int32(variant.Price) * item.Quantity
		subtotal += int(itemSubtotal)

		logItem.Debug(
			"item calculated",
			zap.String("variant_name", variant.Name),
			zap.String("product_name", product.Name),
			zap.Int("price", int(variant.Price)),
			zap.Int32("item_subtotal", itemSubtotal),
		)

		items = append(items, CheckoutSessionItem{
			ID:           uuid.New(),
			VariantID:    variant.ID,
			VariantName:  variant.Name,
			ProductName:  product.Name,
			Quantity:     int(item.Quantity),
			QuantityType: variant.QuantityType,
			ImageURL:     &variant.ImageURL,
			Price:        int(variant.Price),
			Subtotal:     int(itemSubtotal),
		})
	}

	// 2. Calculate fees
	tax := subtotal * 10 / 100
	shippingFee := 0
	discount := 0
	totalPrice := subtotal + tax + shippingFee - discount

	log.Info(
		"price calculated",
		zap.Int("subtotal", subtotal),
		zap.Int("tax", tax),
		zap.Int("shipping_fee", shippingFee),
		zap.Int("discount", discount),
		zap.Int("total_price", totalPrice),
	)

	sessionID := uuid.New()
	sessionExternalID := utils.ExternalIDFromSession("ck", sessionID.String())
	uid := int32(userId)

	// 3. Create session model
	session := &CheckoutSession{
		ID:          sessionID,
		ExternalID:  sessionExternalID,
		UserID:      &uid,
		Status:      CheckoutSessionStatusPending,
		Subtotal:    subtotal,
		Tax:         tax,
		ShippingFee: shippingFee,
		Discount:    discount,
		TotalPrice:  totalPrice,
		ExpiresAt:   time.Now().Add(30 * time.Minute),
	}

	log = log.With(
		zap.String("session_id", session.ID.String()),
		zap.String("status", string(session.Status)),
	)

	// 4. Persist in transaction
	if err := s.repo.CreateCheckoutSession(ctx, session, items); err != nil {
		log.Error(
			"failed to create checkout session",
			zap.Error(err),
		)
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
		zap.String("external_id", externalID),
		zap.String("address_id", addressID),
	)

	log.Info("update session address started")

	session, err := s.repo.GetCheckoutSession(ctx, externalID)
	if err != nil {
		log.Error("failed to get checkout session", zap.Error(err))
		return err
	}

	userID, _ := utils.GetUserIDFromContext(ctx)

	if guestID != nil {
		guestUUID, err := uuid.Parse(*guestID)
		if err != nil {
			log.Warn("invalid guest id format", zap.String("guest_id", *guestID), zap.Error(err))
			return errors.New("invalid guest id")
		}
		if session.GuestID == nil || *session.GuestID != guestUUID {
			log.Warn("forbidden: guest ID mismatch")
			return errors.New("forbidden: guest ID mismatch")
		}
	} else {
		if session.UserID == nil || *session.UserID != int32(userID) {
			log.Warn("forbidden: cannot update others' sessions",
				zap.Int32("session_user_id", *session.UserID),
				zap.Uint("request_user_id", userID),
			)
			return errors.New("forbidden: cannot update others' sessions")
		}
	}

	if session.Status != CheckoutSessionStatusPending {
		log.Warn("checkout session is not editable", zap.String("status", string(session.Status)))
		return errors.New("checkout session is not editable")
	}

	if time.Now().After(session.ExpiresAt) {
		log.Warn("checkout session expired", zap.Time("expires_at", session.ExpiresAt))
		return errors.New("checkout session expired")
	}

	address, err := s.repo.GetUserAddress(ctx, addressID, userID)
	if err != nil {
		log.Error("failed to get user address", zap.Error(err))
		return err
	}

	// 4. Recalculate pricing
	shippingFee := s.calculateShippingFee(address, session.Items)
	tax := s.calculateTax(address, session.Subtotal)

	session.AddressID = &address.ID
	session.ShippingFee = shippingFee
	session.Tax = tax
	session.TotalPrice = session.Subtotal + tax + shippingFee - session.Discount

	// 5. Persist changes
	if err := s.repo.UpdateSessionAddressAndPricing(ctx, session); err != nil {
		log.Error("failed to update session address and pricing", zap.Error(err))
		return err
	}

	log.Info("session address updated successfully")
	return nil
}

func (s *service) UpdateSessionPaymentMethod(
	ctx context.Context,
	externalID string,
	paymentMethod payment.ChannelCode,
	guestID *string,
) error {
	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "UpdateSessionPaymentMethod"),
		zap.String("external_id", externalID),
		zap.String("payment_method", string(paymentMethod)),
	)

	log.Info("update session payment method started")

	switch paymentMethod {
	case payment.MethodBCAVA,
		payment.MethodBNIVA,
		payment.MethodMandiriVA,
		payment.MethodQRIS,
		payment.MethodCOD,
		payment.MethodOVO,
		payment.MethodDANA,
		payment.MethodLINKAJA,
		payment.MethodSHOPEE,
		payment.MethodGOPAY,
		payment.MethodAlfamart,
		payment.MethodIndomaret,
		payment.MethodCreditCard:
		// valid
	default:
		log.Warn("invalid payment method", zap.String("payment_method", string(paymentMethod)))
		return fmt.Errorf("invalid payment method: %s", paymentMethod)
	}

	session, err := s.repo.GetCheckoutSession(ctx, externalID)
	if err != nil {
		log.Error("failed to get checkout session", zap.Error(err))
		return err
	}

	userID, _ := utils.GetUserIDFromContext(ctx)

	if guestID != nil {
		guestUUID, err := uuid.Parse(*guestID)
		if err != nil {
			log.Warn("invalid guest id format", zap.String("guest_id", *guestID), zap.Error(err))
			return errors.New("invalid guest id")
		}
		if session.GuestID == nil || *session.GuestID != guestUUID {
			log.Warn("forbidden: guest ID mismatch")
			return errors.New("forbidden: guest ID mismatch")
		}
	} else {
		if session.UserID == nil || *session.UserID != int32(userID) {
			log.Warn("forbidden: cannot update others' sessions",
				zap.Int32("session_user_id", *session.UserID),
				zap.Uint("request_user_id", userID),
			)
			return errors.New("forbidden: cannot update others' sessions")
		}
	}

	if session.Status != CheckoutSessionStatusPending {
		log.Warn("checkout session is not editable", zap.String("status", string(session.Status)))
		return errors.New("checkout session is not editable")
	}

	if time.Now().After(session.ExpiresAt) {
		log.Warn("checkout session expired", zap.Time("expires_at", session.ExpiresAt))
		return errors.New("checkout session expired")
	}

	// Persist changes
	if err := s.repo.UpdateSessionPaymentMethod(ctx, session.ID, paymentMethod); err != nil {
		log.Error("failed to update session payment method", zap.Error(err))
		return err
	}

	log.Info("session payment method updated successfully")
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
	if session.UserID != nil && *session.UserID != int32(userID) {
		log.Warn("ownership check failed",
			zap.Int32("session_user_id", *session.UserID),
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

	// Idempotency check: see if an order already exists for this session.
	// This handles retries if the payment gateway call fails after order creation.
	order, err := s.repo.GetOrderBySessionID(ctx, session.ID)
	if err != nil {
		// This is an actual DB error, not a "not found" case.
		log.Error("failed to check for existing order by session ID", zap.Error(err))
		return nil, err
	}

	var externalOrderID string

	if order == nil {
		// Order does not exist, this is the first attempt.
		log.Info("creating new order for session")
		externalOrderID = utils.ExternalIDFromSession("pay", session.ID.String())

		order = &Order{
			UserID:      session.UserID,
			TotalAmount: uint(session.TotalPrice),
			Currency:    session.Currency,
			Status:      OrderStatus(model.OrderStatusPendingPayment),
			ExternalID:  externalOrderID,
		}

		if err := s.repo.CreateOrderTx(ctx, order, session); err != nil {
			log.Error("failed to create order in transaction", zap.Error(err))
			return nil, err
		}

		if err := s.repo.ConfirmCheckoutSession(ctx, session); err != nil {
			log.Error("failed to confirm checkout session", zap.Error(err))
			// Note: At this point, an order exists but the session isn't marked as confirmed.
			// The idempotency check at the start of this function will handle retries correctly.
			return nil, err
		}
	} else {
		// Order already exists, this is a retry.
		log.Info("order already exists for this session, retrying payment process", zap.Int32("order_id", order.ID))
		externalOrderID = order.ExternalID
	}

	// 7. Process payment
	_, err = s.OrderToPaymentProcess(ctx, session, externalOrderID, uint(order.ID))
	if err != nil {
		log.Error("failed to process order to payment", zap.Error(err))
		return nil, err
	}

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
		if *session.UserID != int32(userID) {
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

	// Ownership check
	if order.UserID != nil {
		if !ok {
			return nil, errors.New("forbidden")
		}
		if *order.UserID != int32(userID) {
			return nil, errors.New("forbidden")
		}
	}

	paymentData, err := s.paymentRepo.GetPaymentByOrder(ctx, uint(order.ID))
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

func (s *service) GetOrderForWebhook(
	ctx context.Context,
	externalID string,
) (*Order, error) {
	return s.repo.GetOrderByExternalID(ctx, externalID)
}
