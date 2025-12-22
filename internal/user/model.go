package user

type Role string

const (
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"
)

type User struct {
	ID       int
	Email    string
	Password string
	Role     Role
	SellerID *string
}
