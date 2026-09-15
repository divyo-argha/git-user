package components

import (
	"testing"
	"time"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/validate"
)

func TestTokenBadgeState(t *testing.T) {
	if hasToken, expiring, expired := TokenBadgeState(""); hasToken || expiring || expired {
		t.Errorf("expected no badge for empty expiry, got hasToken=%v expiring=%v expired=%v", hasToken, expiring, expired)
	}

	past := time.Now().AddDate(0, 0, -3).Format(validate.DateLayout)
	if hasToken, expiring, expired := TokenBadgeState(past); !hasToken || !expired || expiring {
		t.Errorf("expected expired badge for past date, got hasToken=%v expiring=%v expired=%v", hasToken, expiring, expired)
	}

	soon := time.Now().AddDate(0, 0, 5).Format(validate.DateLayout)
	if hasToken, expiring, expired := TokenBadgeState(soon); !hasToken || !expiring || expired {
		t.Errorf("expected expiring badge for near date, got hasToken=%v expiring=%v expired=%v", hasToken, expiring, expired)
	}

	farOut := time.Now().AddDate(0, 0, 90).Format(validate.DateLayout)
	if hasToken, expiring, expired := TokenBadgeState(farOut); !hasToken || expiring || expired {
		t.Errorf("expected healthy badge for far-out date, got hasToken=%v expiring=%v expired=%v", hasToken, expiring, expired)
	}
}

func TestBuildIdentityItemsIncludesTokenBadge(t *testing.T) {
	soon := time.Now().AddDate(0, 0, 5).Format(validate.DateLayout)
	store := &config.Store{
		Current: "work",
		Users: []config.User{
			{Name: "work", Email: "work@example.com", HTTPSTokenExpiresAt: soon},
		},
	}
	items := buildIdentityItems(store)
	if len(items) < 1 {
		t.Fatal("expected at least one item")
	}
	if !items[0].HasToken || !items[0].TokenExpiring {
		t.Errorf("expected first item to have an expiring token badge, got %#v", items[0])
	}
}
