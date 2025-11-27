package cart

import (
	"database/sql"
	"errors"
	"fmt"
	"warimas-be/internal/graph/model"
)

type Repository interface {
	AddToCart(userID, productID uint, quantity uint) (*CartItem, error)
	GetCart(userID uint,
		filter *model.CartFilterInput,
		sort *model.CartSortInput,
		limit, offset *int32) ([]CartItem, error)
	UpdateCartQuantity(userID, productID uint, quantity int) error
	RemoveFromCart(userID, productID uint) error
	ClearCart(userId uint) error
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) AddToCart(userID, productID uint, quantity uint) (*CartItem, error) {
	// 1️⃣ Check if product exists
	var p model.Product
	err := r.db.QueryRow(`
		SELECT id, name, price, stock 
		FROM products 
		WHERE id = $1
	`, productID).Scan(&p.ID, &p.Name, &p.Price, &p.Stock)
	if err == sql.ErrNoRows {
		return nil, errors.New("product not found")
	}
	if err != nil {
		return nil, err
	}

	// 2️⃣ Check if already in cart
	var existingQty int
	err = r.db.QueryRow(`
		SELECT quantity FROM carts 
		WHERE user_id = $1 AND product_id = $2
	`, userID, productID).Scan(&existingQty)

	switch err {
	case sql.ErrNoRows:
		// 3️⃣ Not in cart → insert new item
		_, err = r.db.Exec(`
			INSERT INTO carts (user_id, product_id, quantity) 
			VALUES ($1, $2, $3)
		`, userID, productID, quantity)
		if err != nil {
			return nil, err
		}

	case nil:
		// 4️⃣ Already in cart → update quantity
		newQty := existingQty + int(quantity)
		_, err = r.db.Exec(`
			UPDATE carts 
			SET quantity = $1, updated_at = NOW() 
			WHERE user_id = $2 AND product_id = $3
		`, newQty, userID, productID)
		if err != nil {
			return nil, err
		}

	default:
		return nil, err
	}

	// 5️⃣ Return the full CartItem (joined with product)
	var ci CartItem
	err = r.db.QueryRow(`
		SELECT 
			c.id, c.user_id, c.product_id, c.quantity, c.created_at, c.updated_at,
			p.id, p.name, p.price, p.stock
		FROM carts c
		JOIN products p ON c.product_id = p.id
		WHERE c.user_id = $1 AND c.product_id = $2
	`, userID, productID).Scan(
		&ci.ID,
		&ci.UserID,
		&ci.ProductID,
		&ci.Quantity,
		&ci.CreatedAt,
		&ci.UpdatedAt,
		&ci.Product.ID,
		&ci.Product.Name,
		&ci.Product.Price,
		&ci.Product.Stock,
	)

	if err != nil {
		return nil, err
	}

	return &ci, nil
}

func (r *repository) GetCart(
	userID uint,
	filter *model.CartFilterInput,
	sort *model.CartSortInput,
	limit, offset *int32,
) ([]CartItem, error) {

	query := `
		SELECT 
			c.id, c.user_id, c.product_id, c.quantity, c.created_at, c.updated_at,
			p.id, p.name, p.price, p.stock
		FROM carts c
		JOIN products p ON c.product_id = p.id
		WHERE c.user_id = $1
	`
	args := []interface{}{userID}
	argPos := 2

	// 🔍 Filters
	if filter != nil {
		if filter.Search != nil && *filter.Search != "" {
			query += fmt.Sprintf(" AND p.name ILIKE $%d", argPos)
			args = append(args, "%"+*filter.Search+"%")
			argPos++
		}
		if filter.InStock != nil {
			if *filter.InStock {
				query += " AND p.stock > 0"
			} else {
				query += " AND p.stock = 0"
			}
		}
	}

	// 🔽 Sorting
	if sort != nil {
		field := "c.created_at"
		switch sort.Field {
		case model.CartSortFieldName:
			field = "p.name"
		case model.CartSortFieldPrice:
			field = "p.price"
		case model.CartSortFieldCreatedAt:
			field = "c.created_at"
		}

		dir := "ASC"
		if sort.Direction == model.SortDirectionDesc {
			dir = "DESC"
		}

		query += fmt.Sprintf(" ORDER BY %s %s", field, dir)
	} else {
		query += " ORDER BY c.created_at DESC"
	}

	// ⏳ Pagination
	if limit != nil {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, *limit)
		argPos++
	}
	if offset != nil {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, *offset)
		argPos++
	}

	// 🧭 Query execution
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query cart: %w", err)
	}
	defer rows.Close()

	var cartItems []CartItem
	for rows.Next() {
		var ci CartItem
		err := rows.Scan(
			&ci.ID,
			&ci.UserID,
			&ci.ProductID,
			&ci.Quantity,
			&ci.CreatedAt,
			&ci.UpdatedAt,
			&ci.Product.ID,
			&ci.Product.Name,
			&ci.Product.Price,
			&ci.Product.Stock,
		)
		if err != nil {
			return nil, err
		}
		cartItems = append(cartItems, ci)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return cartItems, nil
}

func (r *repository) UpdateCartQuantity(userID, productID uint, quantity int) error {
	// Validate that quantity is positive
	if quantity <= 0 {
		return errors.New("quantity must be greater than zero")
	}

	// Update the cart item’s quantity
	res, err := r.db.Exec(`
		UPDATE carts
		SET quantity = $1, updated_at = NOW()
		WHERE user_id = $2 AND product_id = $3
	`, quantity, userID, productID)
	if err != nil {
		return err
	}

	// Check if any rows were actually updated
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("no matching cart item found to update")
	}

	return nil
}

func (r *repository) RemoveFromCart(userID, productID uint) error {
	res, err := r.db.Exec(`
		DELETE FROM carts
		WHERE user_id = $1 AND product_id = $2
	`, userID, productID)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("no matching cart item found to delete")
	}

	return nil
}

func (r *repository) ClearCart(userID uint) error {
	res, err := r.db.Exec(`DELETE FROM carts
	 WHERE user_id=$1`, userID)
	if err != nil {
		return nil
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("no matching cart item found to delete")
	}

	return nil
}
