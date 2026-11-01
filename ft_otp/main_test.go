package main

import "testing"

func TestHOTP(t *testing.T) {
	secret := []byte("12345678901234567890")

	expected := []string{
		"755224",
		"287082",
		"359152",
		"969429",
		"338314",
		"254676",
		"287922",
		"162583",
		"399871",
		"520489",
	}
	for counter, want := range expected {
		got, err := hotp(secret, uint64(counter))

		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf(
				"counter %d: expected %s, got %s",
				counter, want, got,
			)
		}
	}
}
