package auth

type RegisterRequest struct {
	email string
	userName *string
	firstName string
	lastName *string
	phoneNumber *string
	nativeLang  string
	password string
}
