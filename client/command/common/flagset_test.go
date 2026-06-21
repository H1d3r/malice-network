package common

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/spf13/cobra"
)

func TestEncryptionFlagSetDefaults(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	EncryptionFlagSet(cmd.Flags())

	parser, encryption := ParseEncryptionFlags(cmd)
	if parser != "default" {
		t.Fatalf("parser = %q, want default", parser)
	}
	if len(encryption) != 2 {
		t.Fatalf("encryption count = %d, want 2", len(encryption))
	}
	if encryption[0].GetType() != consts.CryptorAES || encryption[0].GetKey() != "maliceofinternal" {
		t.Fatalf("encryption = %#v, want aes/maliceofinternal", encryption[0])
	}
	if encryption[1].GetType() != consts.CryptorXOR || encryption[1].GetKey() != "maliceofinternal" {
		t.Fatalf("encryption = %#v, want xor/maliceofinternal", encryption[1])
	}
}

func TestEncryptionFlagSetAllowsPartialOverrideWithDefaults(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	EncryptionFlagSet(cmd.Flags())
	if err := cmd.ParseFlags([]string{"--encryption-key", "custom-key"}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	_, encryption := ParseEncryptionFlags(cmd)
	if len(encryption) != 2 {
		t.Fatalf("encryption count = %d, want 2", len(encryption))
	}
	if encryption[0].GetType() != consts.CryptorAES || encryption[0].GetKey() != "custom-key" {
		t.Fatalf("encryption = %#v, want aes/custom-key", encryption[0])
	}
	if encryption[1].GetType() != consts.CryptorXOR || encryption[1].GetKey() != "custom-key" {
		t.Fatalf("encryption = %#v, want xor/custom-key", encryption[1])
	}
}

func TestEncryptionFlagSetPairsTypeAndKeyLists(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	EncryptionFlagSet(cmd.Flags())
	if err := cmd.ParseFlags([]string{"--encryption-type", "AES,XOR", "--encryption-key", "aes-key,xor-key"}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	_, encryption := ParseEncryptionFlags(cmd)
	if len(encryption) != 2 {
		t.Fatalf("encryption count = %d, want 2", len(encryption))
	}
	if encryption[0].GetType() != consts.CryptorAES || encryption[0].GetKey() != "aes-key" {
		t.Fatalf("encryption = %#v, want AES/aes-key", encryption[0])
	}
	if encryption[1].GetType() != consts.CryptorXOR || encryption[1].GetKey() != "xor-key" {
		t.Fatalf("encryption = %#v, want XOR/xor-key", encryption[1])
	}
}
