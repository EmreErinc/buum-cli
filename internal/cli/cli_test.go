package cli

import (
	"testing"

	"github.com/emreerinc/buum/pkg/buum"
)

func TestVersionNotEmpty(t *testing.T) {
	if Version == "" {
		t.Fatal("Version must not be empty")
	}
}

func TestNeedsSudo(t *testing.T) {
	plain := buum.Manager{Name: "brew", Steps: [][]string{{"brew", "upgrade"}}}
	sudo := buum.Manager{Name: "port", Steps: [][]string{{"sudo", "port", "selfupdate"}}}
	if needsSudo(buum.Plan{Managers: []buum.Manager{plain}}) {
		t.Error("plan without sudo steps reported needing sudo")
	}
	if !needsSudo(buum.Plan{Managers: []buum.Manager{plain, sudo}}) {
		t.Error("plan with a sudo step not detected")
	}
}
