package auth

type RegisterRequest struct {
	Email       string  `json:"email" validate:"required,email"`
	UserName    *string `json:"user_name,omitempty"`
	FirstName   string  `json:"first_name" validate:"required"`
	LastName    *string `json:"last_name,omitempty"`
	PhoneNumber *string `json:"phone_number,omitempty"`
	NativeLang  string  `json:"native_lang" validate:"required"`
	Password    string  `json:"password" validate:"required,min=8"`
}
