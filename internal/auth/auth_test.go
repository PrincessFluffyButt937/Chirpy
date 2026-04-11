package auth

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCreatePassword(t *testing.T) {
	p_1 := "Hello World"
	h_p1, err := HashPassword(p_1)
	if err != nil {
		t.Errorf("Password hash failed: %s\n", err)
	}
	if p_1 == h_p1 {
		t.Errorf("Password hash failed:\n input: %s\noutput: %s\n", p_1, h_p1)
	}
	match, err := CheckPasswordHash(p_1, h_p1)
	if err != nil {
		t.Errorf("Password check failed: %s\n", err)
	}
	if !match {
		t.Errorf("Passwords do not match\n")
	}
	fmt.Printf("Input password: %s\n", p_1)
	fmt.Printf("Hashed password: %s\n", h_p1)
}

func TestCreatePassword_1(t *testing.T) {
	p_1 := "Hello World"
	h_p1, err := HashPassword(p_1)
	if err != nil {
		t.Errorf("Password hash failed: %s\n", err)
	}
	if p_1 == h_p1 {
		t.Errorf("Password hash failed:\n input: %s\noutput: %s\n", p_1, h_p1)
	}
	match, err := CheckPasswordHash("Hello World1", h_p1)
	if err != nil {
		t.Errorf("Password check failed: %s\n", err)
	}
	if match {
		t.Errorf("Passwords are matching\n")
	}
	fmt.Printf("Input password: %s\n", p_1)
	fmt.Printf("Hashed password: %s\n", h_p1)
}

func TestCreatePassword_2(t *testing.T) {
	p_1 := ""
	_, err := HashPassword(p_1)
	if err == nil {
		t.Errorf("Password lenght check failed")
	}
	if err.Error() != "Password needs to be at least 4 characters." {
		t.Errorf("Unexpected error: %s", err)
	}
}

// JWT testing

func TestJWT_1(t *testing.T) {
	input_id := uuid.New()
	token_secret := "SuP3r_Dup3r_s3cr3t_string*"
	token_str, err := MakeJWT(input_id, token_secret, 1*time.Hour)
	if err != nil {
		t.Errorf("MakeJWT err: %s\n", err)
	}
	output_id, err := ValidateJWT(token_str, token_secret)
	if err != nil {
		t.Errorf("ValidateJWT err: %s\n", err)
	}
	if output_id != input_id {
		t.Errorf("UUIDs are not matching.\n Exp: %v\n Got: %v\n", input_id, output_id)
	}
}

func TestJWT_2(t *testing.T) {
	input_id := uuid.New()
	token_secret := "SuP3r_Dup3r_s3cr3t_string*"
	token_str, err := MakeJWT(input_id, token_secret, 1*time.Hour)
	if err != nil {
		t.Errorf("MakeJWT err: %s\n", err)
	}
	output_id, err := ValidateJWT(token_str, "Definitely_not_SuP3r_Dup3r_s3cr3t_string*")
	if err == nil {
		t.Errorf("ValidateJWT no error was thrown upon diffrent tokenSecret input\n")
	}
	if output_id == input_id {
		t.Errorf("UUIDs should not be matching.\n Exp: %v\n Got: %v\n", input_id, output_id)
	}
}

func TestJWT_3(t *testing.T) {
	input_id := uuid.New()
	token_secret := "SuP3r_Dup3r_s3cr3t_string*"
	token_str, err := MakeJWT(input_id, token_secret, 1*time.Second)
	if err != nil {
		t.Errorf("MakeJWT err: %s\n", err)
	}
	time.Sleep(2 * time.Second)
	output_id, err := ValidateJWT(token_str, token_secret)
	if err == nil {
		t.Errorf("ValidateJWT err: token should have expired\n")
	}
	if output_id == input_id {
		t.Errorf("UUIDs should not be matching.\n Exp: %v\n Got: %v\n", input_id, output_id)
	}
}
