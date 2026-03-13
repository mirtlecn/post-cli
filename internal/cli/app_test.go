package cli

import "testing"

func TestParseNewOptions(t *testing.T) {
	options, err := parseNewOptions([]string{"-s", "demo", "-t", "15", "-u", "-x", "-c", "text", "hello", "world"})
	if err != nil {
		t.Fatalf("parseNewOptions returned error: %v", err)
	}
	if options.Slug != "demo" {
		t.Fatalf("unexpected slug: %s", options.Slug)
	}
	if options.TTL == nil || *options.TTL != 15 {
		t.Fatalf("unexpected ttl: %v", options.TTL)
	}
	if options.Convert != "text" {
		t.Fatalf("unexpected convert: %s", options.Convert)
	}
	if !options.Export {
		t.Fatal("expected export flag")
	}
	if len(options.Args) != 2 {
		t.Fatalf("unexpected args: %#v", options.Args)
	}
}

func TestParseNewOptionsRejectsInvalidConvert(t *testing.T) {
	_, err := parseNewOptions([]string{"-c", "bad"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestShouldPrependNew(t *testing.T) {
	if !shouldPrependNew(nil) {
		t.Fatal("expected prepend for empty args")
	}
	if shouldPrependNew([]string{"ls"}) {
		t.Fatal("did not expect prepend for subcommand")
	}
	if !shouldPrependNew([]string{"hello"}) {
		t.Fatal("expected prepend for free text")
	}
}
