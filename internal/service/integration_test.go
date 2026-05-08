//go:build integration

package service

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	baseURL    = "http://127.0.0.1:8888"
	accrualURL = "http://127.0.0.1:8080"
)

func waitAccrualReady(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(accrualURL + "/api/orders/1")
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("accrual service not available at %s", accrualURL)
}

func randomValidOrderNumber(t *testing.T) string {
	t.Helper()
	for range 50 {
		var b strings.Builder
		for i := 0; i < 11; i++ {
			n, err := rand.Int(rand.Reader, big.NewInt(10))
			if err != nil {
				t.Fatal(err)
			}
			c := byte('0' + n.Int64())
			if i == 0 && c == '0' {
				c = '1'
			}
			b.WriteByte(c)
		}
		body := b.String()
		for d := 0; d <= 9; d++ {
			candidate := body + strconv.Itoa(d)
			if isValidOrderNumber(candidate) {
				return candidate
			}
		}
	}
	t.Fatal("could not generate Luhn-valid order number")
	return ""
}

func TestIntegrationWithAccrual(t *testing.T) {
	waitAccrualReady(t)

	login := fmt.Sprintf("it_user_%d", time.Now().UnixNano())
	t.Run("Register and get token", func(t *testing.T) {
		registerReq := map[string]string{
			"login":    login,
			"password": "testpass123",
		}
		registerJSON, _ := json.Marshal(registerReq)

		resp, err := http.Post(baseURL+"/api/user/register", "application/json", bytes.NewBuffer(registerJSON))
		if err != nil {
			t.Fatalf("Failed to register: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
		}

		var authResp map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		token := authResp["token"]
		if token == "" {
			t.Fatal("Expected token in response")
		}

		orderNum := randomValidOrderNumber(t)

		t.Run("Upload order and check accrual", func(t *testing.T) {
			req, err := http.NewRequest("POST", baseURL+"/api/user/orders", strings.NewReader(orderNum))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "text/plain")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Failed to upload order: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("Expected status 202 or 200, got %d: %s", resp.StatusCode, string(body))
			}

			time.Sleep(3 * time.Second)

			t.Run("Check order status", func(t *testing.T) {
				req, err := http.NewRequest("GET", baseURL+"/api/user/orders", nil)
				if err != nil {
					t.Fatalf("Failed to create request: %v", err)
				}
				req.Header.Set("Authorization", "Bearer "+token)

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatalf("Failed to get orders: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					body, _ := io.ReadAll(resp.Body)
					t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
				}

				var orders []map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&orders); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if len(orders) == 0 {
					t.Fatal("Expected at least one order")
				}

				order := orders[0]
				if order["status"] == "PROCESSED" {
					accrual, ok := order["accrual"]
					if !ok || accrual == nil {
						t.Log("Order processed but no accrual (might be 0)")
					} else {
						t.Logf("Order processed with accrual: %v", accrual)
					}
				}
			})

			t.Run("Check balance", func(t *testing.T) {
				req, err := http.NewRequest("GET", baseURL+"/api/user/balance", nil)
				if err != nil {
					t.Fatalf("Failed to create request: %v", err)
				}
				req.Header.Set("Authorization", "Bearer "+token)

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatalf("Failed to get balance: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					body, _ := io.ReadAll(resp.Body)
					t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
				}

				var balance map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&balance); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				t.Logf("Balance: %v", balance)
			})
		})
	})
}

func TestIntegrationAccrualDirect(t *testing.T) {
	waitAccrualReady(t)

	t.Run("Get accrual for valid order", func(t *testing.T) {
		resp, err := http.Get(accrualURL + "/api/orders/9278923470")
		if err != nil {
			t.Fatalf("Failed to get accrual: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNoContent {
			t.Log("accrual: order not registered yet (204), skip body checks")
			return
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 200 or 204, got %d: %s", resp.StatusCode, string(body))
		}

		var accrualResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&accrualResp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		t.Logf("Accrual response: %v", accrualResp)

		status, ok := accrualResp["status"].(string)
		if !ok {
			t.Fatal("Expected status in response")
		}
		if status != "REGISTERED" && status != "PROCESSING" && status != "PROCESSED" && status != "INVALID" {
			t.Errorf("Unexpected status: %s", status)
		}
	})

	t.Run("Get accrual for invalid order", func(t *testing.T) {
		resp, err := http.Get(accrualURL + "/api/orders/1234567890")
		if err != nil {
			t.Fatalf("Failed to get accrual: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 204 or 200, got %d: %s", resp.StatusCode, string(body))
		}
	})
}
