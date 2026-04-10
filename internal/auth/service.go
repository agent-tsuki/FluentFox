package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/jackc/pgx/v5"
)


type AuthService struct {
	userRepo *users.Repository
	argon  Argon2Config
	tokenService *TokenService
	

}

type TokenService struct {
	userRepo *users.Repository
	argon  Argon2Config
}


func (s *AuthService) registerUser(ctx context.Context, req RegisterRequest) error{
	// validating if email and username allergy exist in db
	err := s.validateUserCredential(ctx, &req)
	if err != nil{
		return err
	}

	// Generate and Hash token 
	hashedData, err := s.tokenService.generateAuthToken()
	if err != nil {
		return err
	}

	hashPassword, err := s.argon.hashedString(req.Password)
	if err != nil{
		return err
	}

	// Start Transaction
	tx, err := s.userRepo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	// Create user data
	createUserErr := s.createUser(ctx, tx, hashPassword, hashedData["hashed"], req)
	if createUserErr != nil{
		// log here
		return createUserErr
	}

	// commit changes to db
	err = tx.Commit(ctx)
	if err != nil{
		return  err
	}


	// send mail this place holder actual mail service yet to register
	mailServer()

	return nil

}

func (s *AuthService) createUser(ctx context.Context, tx pgx.Tx, hashPassword string, hashedToken string, req RegisterRequest) error{
	// create user 
	userID, err := s.userRepo.CreateUser(ctx, tx, req.Email, *req.UserName, hashPassword, req.PhoneNumber)
	if err != nil{
		return err
	}

	// create profile
	// TODO: newNativeLang is place holder if user not provide we will get from location
	newNativeLang := "india" 
	profileErr := s.userRepo.CreateProfile(ctx, tx, userID, req.FirstName, req.LastName, newNativeLang)
	if profileErr != nil{
		return profileErr
	}

	// create use settings
	settingErr := s.userRepo.CreateUsersSettings(ctx, tx, userID, nil, nil)
	if settingErr != nil{
		return settingErr
	}

	// create verification
	tokenExpireAt := s.tokenService.authTokenExpireTime()
	verificationErr := s.userRepo.CreateUserVerification(ctx, tx, userID, hashedToken, &tokenExpireAt)
	if verificationErr != nil{
		return verificationErr
	}

	return nil
}


func (s *AuthService) validateUserCredential(ctx context.Context, req *RegisterRequest) error {
	// check user email
	exist, err := s.userRepo.GetExistingUserForEmail(ctx, req.Email)

	if err != nil {
		return exceptions.Wrap(500, "INTERNAL_ERROR", "error while fetching email", err)
	}

	if exist {
		return exceptions.ErrEmailAlreadyInUse
	}

	// check user name if given
	if req.UserName != nil {
		exist, err := s.userRepo.GetExistingUserForUsername(ctx, *req.UserName)
		if err != nil {
			return exceptions.Wrap(500, "INTERNAL_ERROR", "error while fetching username", err)
		}

		if exist {
			return exceptions.ErrUsernameAlreadyInUse
		}
	} else {
		generatedUserName, err := s.getGenerateUsername(ctx, *req)
		if err != nil {
			return errors.New("Error while generating username")
		}
		req.UserName = &generatedUserName
	}

	return nil
}

func (s *AuthService) getGenerateUsername(ctx context.Context, req RegisterRequest) (string, error) {
	for i:=0; i<common.USERNAME_RETRY; i++  {
		newUsername, err := s.generateUsername(req.FirstName, req.LastName)
		if err != nil{
			// Will be here logs
			continue
		}
		exist, err := s.userRepo.GetExistingUserForUsername(ctx, newUsername)
		if err != nil{
			// if we are not able to fetch user name due to error 
			// we will try for some time, before return error
			continue
		}

		if !exist{
			// we will log there
			return newUsername, nil
		}
	}

	return  "", errors.New("Username not generated try again")
}

func (s *AuthService) generateUsername(firstName string, lastName *string) (string, error) {
	suffix, err := generateString(common.USERNAME_SUFFIX_LEN)
	if err != nil {
		return  suffix, err
	}
	if lastName != nil{
		return fmt.Sprintf("%s%s%s", firstName, *lastName, suffix) , nil
	}
    return fmt.Sprintf("%s%s", firstName, suffix) , nil
}

func (t *TokenService) generateAuthToken() (map[string]string, error){
	tokenData := make(map[string]string)
	// generate token
	token, err := generateString(common.AUTH_TOKEN_LEN)
	if err != nil{
		return tokenData, err
	}

	hashToken, err := t.argon.hashedString(token)
	if err != nil{
		return tokenData, err
	}
	tokenData["token"] = token
	tokenData["hashed"] = hashToken
	return  tokenData, err
}

func (t *TokenService) authTokenExpireTime() time.Time {
	// current time
	currentTime := time.Now().UTC()

	// Time when token expire
	return currentTime.Add(time.Duration(common.AUTH_TOKEN_EXPIRE_HOUR) * time.Hour)
}
