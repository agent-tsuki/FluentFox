package auth


type RegisterRequest struct {
	Email       string  `json:"email"        validate:"required,email"   format:"email"   doc:"User's email address"`
	UserName    *string `json:"user_name,omitempty"                                          doc:"Username; auto-generated if omitted"`
	FirstName   string  `json:"first_name"   validate:"required"                             doc:"First name"`
	LastName    *string `json:"last_name,omitempty"                                          doc:"Last name"`
	PhoneNumber *string `json:"phone_number,omitempty"                                       doc:"Phone number"`
	NativeLang  string  `json:"native_lang"  validate:"required"                             doc:"Native language code (e.g. en, ja)"`
	Password    string  `json:"password"     validate:"required,min=8"   minLength:"8"       doc:"Password — minimum 8 characters"`
}


type LoginRequest struct {
	Email       string  `json:"email"        validate:"required,email"   format:"email"   doc:"User's email address"`
	Password    string  `json:"password"     validate:"required,min=8"   minLength:"8"       doc:"Password — minimum 8 characters"`
}

