package cart

import (
	"database/sql"
	"errors"
	"warimas-be/internal/product"
)

type Repository interface {
	AddToCart(userID, productID uint, quantity uint) (*product.Product, error)
	GetCart(userID uint) ([]CartItem, error)
	UpdateCartQuantity(userID, productID uint, quantity int) error
	RemoveFromCart(userID, productID uint) error
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) AddToCart(userID, productID uint, quantity uint) (*product.Product, error) {
	// 1️⃣ Check if product exists
	var p product.Product
	err := r.db.QueryRow(
		"SELECT id, name, price, stock FROM products WHERE id = $1",
		productID,
	).Scan(&p.ID, &p.Name, &p.Price, &p.Stock)

	if err == sql.ErrNoRows {
		return nil, errors.New("product not found")
	}
	if err != nil {
		return nil, err
	}

	// 2️⃣ Check if already in cart
	var existingQty uint
	err = r.db.QueryRow(
		"SELECT quantity FROM carts WHERE user_id = $1 AND product_id = $2",
		userID, productID,
	).Scan(&existingQty)

	switch err {
	case sql.ErrNoRows:
		// 3️⃣ Not in cart → insert
		_, err = r.db.Exec(
			"INSERT INTO carts (user_id, product_id, quantity) VALUES ($1, $2, $3)",
			userID, productID, quantity,
		)
		if err != nil {
			return nil, err
		}
	case nil:
		// 4️⃣ Already exists → update quantity
		newQty := existingQty + quantity
		_, err = r.db.Exec(
			"UPDATE carts SET quantity = $1 WHERE user_id = $2 AND product_id = $3",
			newQty, userID, productID,
		)
		if err != nil {
			return nil, err
		}
	default:
		return nil, err // some other error
	}

	return &p, nil
}

func (r *repository) GetCart(userID uint) ([]CartItem, error) {
	rows, err := r.db.Query(`
        SELECT 
            c.id, c.user_id, c.product_id, c.quantity, c.created_at, c.updated_at,
            p.id, p.name, p.price, p.stock
        FROM carts c
        JOIN products p ON c.product_id = p.id
        WHERE c.user_id = $1
        ORDER BY c.created_at DESC
    `, userID)
	if err != nil {
		return nil, err
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
