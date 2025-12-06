package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// cmdSignup handles user signup
func cmdSignup(args []string) error {
	// Load config to get backend URL
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("config required for signup: %w\nRun 'pipeboard init' to create a config", err)
	}

	if cfg.Sync == nil || cfg.Sync.Backend != "hosted" {
		return fmt.Errorf("signup requires hosted backend to be configured\n\nAdd to ~/.config/pipeboard/config.yaml:\n\nsync:\n  backend: hosted\n  hosted:\n    url: https://pipeboard-mobile-backend.fly.dev\n    email: your@email.com\n  encryption: aes256\n  passphrase: your-encryption-password")
	}

	if cfg.Sync.Hosted == nil || cfg.Sync.Hosted.URL == "" {
		return fmt.Errorf("hosted.url not configured")
	}

	email := cfg.Sync.Hosted.Email
	if email == "" {
		// Prompt for email
		fmt.Print("Email: ")
		reader := bufio.NewReader(os.Stdin)
		email, err = reader.ReadString('\n')
		if err != nil {
			return err
		}
		email = strings.TrimSpace(email)
	}

	// Prompt for password
	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return err
	}
	fmt.Println()
	password := string(passwordBytes)

	// Confirm password
	fmt.Print("Confirm password: ")
	confirmBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return err
	}
	fmt.Println()
	confirm := string(confirmBytes)

	if password != confirm {
		return fmt.Errorf("passwords do not match")
	}

	// Call signup API
	if err := Signup(cfg.Sync.Hosted.URL, email, password); err != nil {
		return err
	}

	fmt.Printf("Account created successfully for %s\n", email)
	fmt.Println("You can now use 'pipeboard push' and 'pipeboard pull' to sync clipboard with your mobile devices")
	return nil
}

// cmdLogin handles user login
func cmdLogin(args []string) error {
	// Load config to get backend URL
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("config required for login: %w\nRun 'pipeboard init' to create a config", err)
	}

	if cfg.Sync == nil || cfg.Sync.Backend != "hosted" {
		return fmt.Errorf("login requires hosted backend to be configured\n\nAdd to ~/.config/pipeboard/config.yaml:\n\nsync:\n  backend: hosted\n  hosted:\n    url: https://pipeboard-mobile-backend.fly.dev\n    email: your@email.com\n  encryption: aes256\n  passphrase: your-encryption-password")
	}

	if cfg.Sync.Hosted == nil || cfg.Sync.Hosted.URL == "" {
		return fmt.Errorf("hosted.url not configured")
	}

	email := cfg.Sync.Hosted.Email
	if email == "" {
		// Prompt for email
		fmt.Print("Email: ")
		reader := bufio.NewReader(os.Stdin)
		email, err = reader.ReadString('\n')
		if err != nil {
			return err
		}
		email = strings.TrimSpace(email)
	}

	// Prompt for password
	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return err
	}
	fmt.Println()
	password := string(passwordBytes)

	// Call login API
	if err := Login(cfg.Sync.Hosted.URL, email, password); err != nil {
		return err
	}

	fmt.Printf("Logged in successfully as %s\n", email)
	fmt.Println("You can now use 'pipeboard push' and 'pipeboard pull' to sync clipboard with your mobile devices")
	return nil
}

// cmdLogout handles user logout
func cmdLogout(args []string) error {
	// Load config to get email
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("config required for logout: %w", err)
	}

	if cfg.Sync == nil || cfg.Sync.Hosted == nil || cfg.Sync.Hosted.Email == "" {
		return fmt.Errorf("no email configured in hosted backend")
	}

	email := cfg.Sync.Hosted.Email

	// Clear token
	if err := Logout(email); err != nil {
		return fmt.Errorf("logout failed: %w", err)
	}

	fmt.Printf("Logged out %s\n", email)
	return nil
}
