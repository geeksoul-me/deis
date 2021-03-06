package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"syscall"

	"github.com/deis/deis/client-go/controller/client"
	"github.com/deis/deis/client-go/controller/models/auth"
	"golang.org/x/crypto/ssh/terminal"
)

// Register creates a account on a Deis controller.
func Register(controller string, username string, password string, email string,
	sslVerify bool) error {

	u, err := url.Parse(controller)
	httpClient := client.CreateHTTPClient(sslVerify)

	if err != nil {
		return err
	}

	controllerURL, err := chooseScheme(*u)

	if err != nil {
		return err
	}

	if err = client.CheckConection(httpClient, controllerURL); err != nil {
		return err
	}

	if username == "" {
		fmt.Print("username: ")
		fmt.Scanln(&username)
	}

	if password == "" {
		fmt.Print("password: ")
		password, err = readPassword()
		fmt.Printf("\npassword (confirm): ")
		passwordConfirm, err := readPassword()
		fmt.Println()

		if err != nil {
			return err
		}

		if password != passwordConfirm {
			return errors.New("Password mismatch, aborting registration.")
		}
	}

	if email == "" {
		fmt.Print("email: ")
		fmt.Scanln(&email)
	}

	c := &client.Client{ControllerURL: controllerURL, SSLVerify: sslVerify, HTTPClient: httpClient}

	tempClient, err := client.New()

	if err == nil {
		c.Token = tempClient.Token
	}

	err = auth.Register(c, username, password, email)

	if err != nil {
		return err
	}

	fmt.Printf("Registered %s\n", username)
	return doLogin(c, username, password)
}

func doLogin(c *client.Client, username, password string) error {
	token, err := auth.Login(c, username, password)

	if err != nil {
		return err
	}

	c.Token = token
	c.Username = username

	err = c.Save()

	if err != nil {
		return nil
	}

	fmt.Printf("Logged in as %s\n", username)
	return nil
}

// Login to a Deis controller.
func Login(controller string, username string, password string, sslVerify bool) error {
	u, err := url.Parse(controller)

	if err != nil {
		return err
	}

	controllerURL, err := chooseScheme(*u)
	httpClient := client.CreateHTTPClient(sslVerify)

	if err != nil {
		return err
	}

	if err = client.CheckConection(httpClient, controllerURL); err != nil {
		return err
	}

	if username == "" {
		fmt.Print("username: ")
		fmt.Scanln(&username)
	}

	if password == "" {
		fmt.Print("password: ")
		password, err = readPassword()
		fmt.Println()

		if err != nil {
			return err
		}
	}

	c := &client.Client{ControllerURL: controllerURL, SSLVerify: sslVerify, HTTPClient: httpClient}

	return doLogin(c, username, password)
}

// Logout from a Deis controller.
func Logout() error {
	if err := client.Delete(); err != nil {
		return err
	}

	fmt.Println("Logged out")
	return nil
}

// Passwd changes a user's password.
func Passwd(username string, password string, newPassword string) error {
	c, err := client.New()

	if err != nil {
		return err
	}

	if password == "" && username == "" {
		fmt.Print("current password: ")
		password, err = readPassword()
		fmt.Println()

		if err != nil {
			return err
		}
	}

	if newPassword == "" {
		fmt.Print("new password: ")
		newPassword, err = readPassword()
		fmt.Printf("\nnew password (confirm): ")
		passwordConfirm, err := readPassword()

		fmt.Println()

		if err != nil {
			return err
		}

		if newPassword != passwordConfirm {
			return errors.New("Password mismatch, not changing.")
		}
	}

	err = auth.Passwd(c, username, password, newPassword)

	if err != nil {
		return err
	}

	fmt.Println("Password change succeeded.")
	return nil
}

// Cancel deletes a user's account.
func Cancel(username string, password string, yes bool) error {
	c, err := client.New()

	if err != nil {
		return err
	}

	fmt.Println("Please log in again in order to cancel this account")

	if err = Login(c.ControllerURL.String(), username, password, c.SSLVerify); err != nil {
		return err
	}

	if yes == false {
		confirm := ""

		c, err = client.New()

		if err != nil {
			return err
		}

		fmt.Printf("cancel account %s at %s? (y/N): ", c.Username, c.ControllerURL.String())
		fmt.Scanln(&confirm)

		if strings.ToLower(confirm) == "y" {
			yes = true
		}
	}

	if yes == false {
		fmt.Println("Account not changed")
		return nil
	}

	err = auth.Delete(c)

	if err != nil {
		return err
	}

	if err := client.Delete(); err != nil {
		return err
	}

	fmt.Println("Account cancelled")
	return nil
}

// Whoami prints the logged in user.
func Whoami() error {
	c, err := client.New()

	if err != nil {
		return err
	}

	fmt.Printf("You are %s at %s\n", c.Username, c.ControllerURL.String())
	return nil
}

// Regenerate regenenerates a user's token.
func Regenerate(username string, all bool) error {
	c, err := client.New()

	if err != nil {
		return err
	}

	token, err := auth.Regenerate(c, username, all)

	if err != nil {
		return err
	}

	if username == "" && all == false {
		c.Token = token

		err = c.Save()

		if err != nil {
			return err
		}
	}

	fmt.Println("Token Regenerated")
	return nil
}

func readPassword() (string, error) {
	password, err := terminal.ReadPassword(int(syscall.Stdin))

	return string(password), err
}

func chooseScheme(u url.URL) (url.URL, error) {
	if u.Scheme == "" {
		u.Scheme = "http"
		u, err := url.Parse(u.String())

		if err != nil {
			return url.URL{}, err
		}

		return *u, nil
	}

	return u, nil
}
