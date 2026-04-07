package auth

import (
	"fmt"
	"testing"
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
