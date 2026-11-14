package database

import (
	"regexp"
	"testing"
)

func TestSeedProductCatalogue(t *testing.T) {
	skuPattern := regexp.MustCompile(`^[A-Z]{2,3}-[A-Z]{3}-[0-9]{3}$`)
	seen := map[string]bool{}
	lowStock := 0
	for _, p := range seedProducts {
		if !skuPattern.MatchString(p.SKU) {
			t.Errorf("product %q has a non-standard SKU", p.SKU)
		}
		if seen[p.SKU] {
			t.Errorf("duplicate SKU %q", p.SKU)
		}
		seen[p.SKU] = true
		if p.Name == "" {
			t.Errorf("product %q has no name", p.SKU)
		}
		if p.Description == "" {
			t.Errorf("product %q has no description", p.SKU)
		}
		if p.Category == "" || p.Unit == "" {
			t.Errorf("product %q is missing category or unit", p.SKU)
		}
		if p.Cost <= 0 {
			t.Errorf("product %q has non-positive cost %.2f", p.SKU, p.Cost)
		}
		if p.MinStock < 0 {
			t.Errorf("product %q has negative minimum stock", p.SKU)
		}
		if p.Qty < 0 {
			t.Errorf("product %q has negative initial quantity", p.SKU)
		}
		if p.MinStock > 0 && p.Qty < p.MinStock {
			lowStock++
		}
	}
	if lowStock < 2 {
		t.Errorf("expected at least 2 low-stock products, got %d", lowStock)
	}
}

func TestSeedUsers(t *testing.T) {
	seen := map[string]bool{}
	for _, u := range seedUsers {
		if seen[u.Email] {
			t.Errorf("duplicate user email %q", u.Email)
		}
		seen[u.Email] = true
		if u.FullName == "" {
			t.Errorf("user %q has no full name", u.Email)
		}
		if len(u.Password) < 8 {
			t.Errorf("user %q password is too short", u.Email)
		}
		if u.Role == "DEPARTMENT_USER" && u.Department == "" {
			t.Errorf("department user %q has no department", u.Email)
		}
	}
	for _, email := range []string{"supervisor@wms.local", "warehouse@wms.local", "staff@wms.local"} {
		if !seen[email] {
			t.Errorf("required demo user %q is missing", email)
		}
	}
}
