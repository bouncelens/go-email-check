package emailcheck_test

import (
	"context"
	"fmt"

	emailcheck "github.com/bouncelens/go-email-check"
)

// Check a list, then keep only the addresses that are worth sending to.
func Example() {
	results := emailcheck.Check(context.Background(), []string{
		"jane@gmial.com",      // typo of gmail.com
		"temp@mailinator.com", // disposable
		"jane@gmail.com",
	})
	for _, r := range results {
		if r.DidYouMean != nil {
			fmt.Println(r.Input, "→ did you mean", *r.DidYouMean)
		}
		if r.Status == emailcheck.StatusUnconfirmed {
			fmt.Println("keep", r.Input)
		}
	}
	fmt.Printf("%+v\n", emailcheck.Summarize(results))
}

// Validate a sign-up form field.
func ExampleCheckOne() {
	r := emailcheck.CheckOne(context.Background(), "jane@outlok.com")
	if r.Status != emailcheck.StatusUnconfirmed {
		fmt.Println("Please check your email address:", r.Reasons)
	}
}
