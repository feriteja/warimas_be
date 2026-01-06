package order

import (
	"context"
	"errors"
	"fmt"
	"log"
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
		sessionID string,
	) (*Order, error)
	CreateOrder(userID uint, userEmail string) (*Order, *payment.PaymentResponse, error)
	GetOrders(ctx context.Context, filter *model.OrderFilterInput, sort *model.OrderSortInput, limit, page *int32) ([]*model.Order, error)
	GetOrderDetail(userID, orderID uint, isAdmin bool) (*Order, error)
	UpdateOrderStatus(orderID uint, status OrderStatus) error
	MarkAsPaid(referenceID string) error
	MarkAsFailed(referenceID string) error
	CreateSession(
		ctx context.Context,
		input model.CreateSessionCheckoutInput,
	) (*CheckoutSession, error)

	UpdateSessionAddress(
		ctx context.Context,
		sessionID string,
		userID uint,
		addressID string,
	) error
	ConfirmSession(
		ctx context.Context,
		sessionID string,
		userID uint,
	) (*CheckoutSession, error)
	GetSession(
		ctx context.Context,
		sessionID string,
		userID *uint,
	) (*CheckoutSession, error)
}

type service struct {
	repo        Repository
	paymentRepo payment.Repository
	paymentGate payment.Gateway
}

func NewService(repo Repository, payRepo payment.Repository, payGate payment.Gateway) Service {
	return &service{
		repo:        repo,
		paymentRepo: payRepo,
		paymentGate: payGate,
	}
}

func (s *service) CreateFromSession(
	ctx context.Context,
	sessionID string,
) (*Order, error) {

	// 1. Load session
	session, err := s.repo.GetCheckoutSession(ctx, sessionID)
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
		UserID:    session.UserID,
		Status:    OrderStatus(model.OrderStatusPending),
		Total:     uint(session.TotalPrice),
		CreatedAt: time.Now(),
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
func (s *service) CreateOrder(userID uint, userEmail string) (*Order, *payment.PaymentResponse, error) {
	if userID == 0 {
		return nil, nil, errors.New("unauthorized")
	}

	order, err := s.repo.CreateOrder(userID)
	if err != nil {
		return nil, nil, err
	}

	var items []payment.OrderItem
	for _, oi := range order.Items {
		items = append(items, payment.OrderItem{
			ProductID: oi.ProductID,
			Quantity:  oi.Quantity,
		})
	}
	payResp, err := s.paymentGate.CreateInvoice(order.ID, userEmail, order.Total, userEmail, items, payment.ChannelBCA)
	if err != nil {
		return order, nil, fmt.Errorf("failed to create payment invoice: %w", err)
	}

	p := &payment.Payment{
		OrderID:       order.ID,
		ExternalID:    payResp.ExternalID,
		InvoiceURL:    payResp.InvoiceURL,
		Amount:        payResp.Amount,
		Status:        payResp.Status,
		PaymentMethod: payResp.PaymentMethod,
		ChannelCode:   payResp.ChannelCode,
		PaymentCode:   payResp.PaymentCode,
	}

	err = s.paymentRepo.SavePayment(p)
	if err != nil {
		return order, nil, fmt.Errorf("failed to save payment: %w", err)
	}

	return order, payResp, nil
}

// ✅ Get list of orders (user or admin)
func (s *service) GetOrders(ctx context.Context, filter *model.OrderFilterInput, sort *model.OrderSortInput, limit, page *int32) ([]*model.Order, error) {

	orders, err := s.repo.GetOrders(ctx, filter, sort, limit, page)
	if err != nil {
		return nil, err
	}

	return orders, nil
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
		StatusAccepted: true,
		StatusRejected: true,
		StatusCanceled: true,
	}

	if !validStatuses[status] {
		return fmt.Errorf("invalid status: %s", status)
	}

	return s.repo.UpdateOrderStatus(orderID, status)
}

func (s *service) MarkAsPaid(referenceID string) error {
	// order, err := s.repo.GetByReferenceID(referenceID)
	// if err != nil {
	// 	return err
	// }

	// if order.Status == "PAID" {
	// 	log.Printf("Order %s already marked as paid", referenceID)
	// 	return nil
	// }

	err := s.repo.UpdateStatusByReferenceID(referenceID, "PAID")
	if err != nil {
		return err
	}

	log.Printf("✅ Order %s marked as PAID", referenceID)
	return nil
}

func (s *service) MarkAsFailed(referenceID string) error {
	order, err := s.repo.GetByReferenceID(referenceID)
	if err != nil {
		return err
	}

	if order.Status == "FAILED" {
		log.Printf("Order %s already marked as failed", referenceID)
		return nil
	}

	err = s.repo.UpdateStatusByReferenceID(referenceID, "FAILED")
	if err != nil {
		return err
	}

	log.Printf("❌ Order %s marked as FAILED", referenceID)
	return nil
}

func (s *service) CreateSession(
	ctx context.Context,
	input model.CreateSessionCheckoutInput,
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

	// 3. Create session model
	session := &CheckoutSession{
		ID:          uuid.New(),
		UserID:      &userId,
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
	sessionID string,
	userID uint,
	addressID string,
) error {

	// 1. Load session
	session, err := s.repo.GetCheckoutSession(ctx, sessionID)
	if err != nil {
		return err
	}

	// 2. Validate session state
	if session.Status != CheckoutSessionStatusPending {
		return errors.New("checkout session is not editable")
	}

	if time.Now().After(session.ExpiresAt) {
		return errors.New("checkout session expired")
	}

	// 3. Validate address ownership
	address, err := s.repo.GetUserAddress(ctx, addressID, userID)
	if err != nil {
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
	return s.repo.UpdateSessionAddressAndPricing(ctx, session)
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
	sessionID string,
	userID uint,
) (*CheckoutSession, error) {

	// 1. Load session (with items)
	session, err := s.repo.GetCheckoutSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// 2. Ownership check (if not guest)
	if session.UserID != nil && *session.UserID != userID {
		return nil, errors.New("forbidden")
	}

	// 3. Validate state
	if session.Status != CheckoutSessionStatusPending {
		return nil, errors.New("checkout session already confirmed")
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, errors.New("checkout session expired")
	}

	if session.AddressID == nil {
		return nil, errors.New("shipping address not set")
	}

	if len(session.Items) == 0 {
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
			return nil, err
		}
		if !ok {
			return nil, errors.New("product out of stock")
		}
	}

	// 5. Lock session
	session.Status = CheckoutSessionStatusPending // still pending, but locked

	// 6. Create payment intent (stub for now)
	// paymentRef, err := s.payment.CreatePaymentIntent(
	// 	ctx,
	// 	session.ID,
	// 	session.TotalPrice,
	// )
	// if err != nil {
	// 	return nil, err
	// }

	// session.PaymentRef = &paymentRef
	session.PaymentRef = nil

	// 7. Persist changes
	err = s.repo.ConfirmCheckoutSession(ctx, session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (s *service) GetSession(
	ctx context.Context,
	sessionID string,
	userID *uint,
) (*CheckoutSession, error) {

	session, err := s.repo.GetCheckoutSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Ownership check (if session is tied to a user)
	if session.UserID != nil && userID != nil {
		if *session.UserID != *userID {
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
