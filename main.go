package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

// Data structures for API responses
type AuthResponse struct {
	User AuthUser `json:"user"`
}

type AuthUser struct {
	UserId string `json:"userId"`
	Token  string `json:"token"`
}

type TimeResponse struct {
	UserCalculatedData []UserCalculatedData `json:"userCalculatedData"`
}

type UserCalculatedData struct {
	Hours      []Hour       `json:"hours"`
	TotalHours []TotalHours `json:"totalHours"`
}

type TotalHours struct {
	Amount float64 `json:"amount"`
}

type Hour struct {
	Amount float64 `json:"amount"`
}

// Configuration constants
const (
	rubixxBaseURL           = "https://rubixx.timetrakgo.com/api"
	expectedWorkHoursPerDay = 7.4
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// run encapsulates the main program flow
func run() error {
	username, password, err := getCredentials()
	if err != nil {
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	userId, token, err := getUserAuthToken(username, password)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	numberOfDaysWorked, numberOfHoursWorked, err := getWorkingHours(userId, token)
	if err != nil {
		return fmt.Errorf("failed to retrieve working hours: %w", err)
	}

	logResult(numberOfDaysWorked, numberOfHoursWorked)
	return nil
}

func getCredentials() (string, string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Please enter your username:")
	username, err := reader.ReadString('\n')
	if err != nil {
		return "", "", fmt.Errorf("failed to read username: %w", err)
	}

	fmt.Println("Please enter your password:")
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", "", fmt.Errorf("failed to read password: %w", err)
	}

	password := string(bytePassword)
	fmt.Println() // Add a newline after password input

	return strings.TrimSpace(username), strings.TrimSpace(password), nil
}

func getUserAuthToken(username string, password string) (string, string, error) {
	rubixxAuthUrl := fmt.Sprintf("%s/auth/authenticate?username=%s&password=%s", rubixxBaseURL, username, password)

	res, err := http.Get(rubixxAuthUrl)
	if err != nil {
		return "", "", fmt.Errorf("authentication request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("authentication failed with status: %s", res.Status)
	}

	var authResponse AuthResponse
	if err := json.NewDecoder(res.Body).Decode(&authResponse); err != nil {
		return "", "", fmt.Errorf("failed to parse authentication response: %w", err)
	}

	return authResponse.User.UserId, authResponse.User.Token, nil
}

func getStartAndEndDates() (string, string) {
	current := time.Now()

	startOfMonth := time.Date(current.Year(), current.Month(), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, -1)

	fmt.Printf("Date range: %s to %s\n",
		startOfMonth.Format("2006-01-02"),
		endOfMonth.Format("2006-01-02"))

	return startOfMonth.Format("2006-01-02"), endOfMonth.Format("2006-01-02")
}

func getWorkingHours(userId string, token string) (int, float64, error) {
	startOfMonth, endOfMonth := getStartAndEndDates()
	rubixxPunchUrl := fmt.Sprintf("%s/punch/GetCalculatedHours?userId=%s&groupId=&StartDateTime=%s&EndDateTime=%s&ReturnWorkWeek=false",
		rubixxBaseURL, userId, startOfMonth, endOfMonth)

	req, err := http.NewRequest("GET", rubixxPunchUrl, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("API returned error status: %s", res.Status)
	}

	var timeResponse TimeResponse
	if err := json.NewDecoder(res.Body).Decode(&timeResponse); err != nil {
		return 0, 0, fmt.Errorf("failed to parse time data: %w", err)
	}

	if len(timeResponse.UserCalculatedData) == 0 ||
		len(timeResponse.UserCalculatedData[0].Hours) == 0 ||
		len(timeResponse.UserCalculatedData[0].TotalHours) == 0 {
		return 0, 0, fmt.Errorf("received empty or invalid time data")
	}

	numberOfDaysWorked := len(timeResponse.UserCalculatedData[0].Hours)
	totalHours := timeResponse.UserCalculatedData[0].TotalHours[0].Amount

	return numberOfDaysWorked, totalHours, nil
}

func logResult(numberOfDaysWorked int, numberOfHoursWorked float64) {
	expectedWorkingHours := expectedWorkHoursPerDay * float64(numberOfDaysWorked)
	delta := numberOfHoursWorked - expectedWorkingHours

	fmt.Printf("Days worked: %d\n", numberOfDaysWorked)
	fmt.Printf("Hours worked: %.2f\n", numberOfHoursWorked)
	fmt.Printf("Expected hours: %.2f\n", expectedWorkingHours)

	if delta >= 0 {
		fmt.Printf("Status: Over by %.2f hours\n", delta)
	} else {
		fmt.Printf("Status: Down by %.2f hours\n", -delta)
	}
}
