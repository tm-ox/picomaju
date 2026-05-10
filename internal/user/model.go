package user

import "golang.org/x/crypto/bcrypt"

type Role string

const (
	RoleOwner   Role = "owner"
	RoleManager Role = "manager"
	RoleStaff   Role = "staff"
)

func (r Role) Label() string {
	switch r {
	case RoleOwner:
		return "Owner"
	case RoleManager:
		return "Manager"
	case RoleStaff:
		return "Staff"
	default:
		return string(r)
	}
}

// User is a human account that can log in to Picomaju.
type User struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Role        Role   `json:"role"`
	PINHash     string `json:"pin_hash"`
	StaffID     string `json:"staff_id,omitempty"`   // optional linked agent profile
	Description string `json:"description,omitempty"` // written into USER.md
}

func (u *User) SetPIN(pin string) error {
	h, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PINHash = string(h)
	return nil
}

func (u *User) CheckPIN(pin string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.PINHash), []byte(pin)) == nil
}

func (u *User) CanManageUsers() bool { return u.Role == RoleOwner }
func (u *User) CanManageSettings() bool { return u.Role == RoleOwner }
func (u *User) CanManageBilling() bool  { return u.Role == RoleOwner }
