package util

import "testing"

func TestHashPassword_ProducesVerifiableHash(t *testing.T) {
	hash, err := HashPassword("correct horse")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == "correct horse" {
		t.Fatal("hash must not equal the plaintext password")
	}
	if err := CheckPassword(hash, "correct horse"); err != nil {
		t.Fatalf("CheckPassword rejected the correct password: %v", err)
	}
}

func TestCheckPassword_WrongPassword(t *testing.T) {
	hash, err := HashPassword("correct horse")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if err := CheckPassword(hash, "wrong horse"); err == nil {
		t.Fatal("CheckPassword accepted a wrong password")
	}
}

func TestHashPassword_SaltsEachHash(t *testing.T) {
	first, err := HashPassword("same password")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	second, err := HashPassword("same password")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if first == second {
		t.Fatal("hashing the same password twice produced identical hashes; salt is missing")
	}
}
