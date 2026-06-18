# GitHub Actions PR Test Workflow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a GitHub Actions workflow that runs unit tests on every PR and, when they pass, runs an end-to-end CRUD cycle against a Spanner emulator.

**Architecture:** One workflow file with two jobs. The first job builds, vets, and unit-tests the Go module. The second job depends on the first, starts a Spanner emulator service container, runs migrations, starts the API server, and executes a Go e2e test package that calls the REST and webhook endpoints.

**Tech Stack:** GitHub Actions, Go 1.26.4, Spanner emulator, chi HTTP router, standard library `net/http` and `testing`.

---

## File structure

| File | Responsibility |
|---|---|
| `.github/workflows/pr-tests.yml` | GitHub Actions workflow definition. |
| `apps/api/e2e/e2e_test.go` | End-to-end test package. Skips unless `GCP_SPANNER_EMULATOR_HOST` is set. |

---

### Task 1: Create the GitHub Actions workflow

**Files:**
- Create: `.github/workflows/pr-tests.yml`

- [ ] **Step 1: Write the workflow file**

  Create `.github/workflows/pr-tests.yml` with this exact content:

  ```yaml
  name: PR Tests

  on:
    pull_request:
      types: [opened, synchronize, reopened]

  jobs:
    unit-tests:
      runs-on: ubuntu-latest
      defaults:
        run:
          working-directory: apps/api
      steps:
        - uses: actions/checkout@v4

        - uses: actions/setup-go@v5
          with:
            go-version-file: apps/api/go.mod

        - name: Build
          run: go build ./...

        - name: Vet
          run: go vet ./...

        - name: Unit test
          run: go test ./...

    e2e:
      runs-on: ubuntu-latest
      needs: unit-tests
      defaults:
        run:
          working-directory: apps/api
      services:
        spanner:
          image: gcr.io/cloud-spanner-emulator/emulator
          ports:
            - 9010:9010
      env:
        GCP_SPANNER_PROJECT_ID: invisible-ms-local
        GCP_SPANNER_INSTANCE_ID: invisible-ms-instance
        GCP_SPANNER_DATABASE_ID: invisible-ms-db
        GCP_SPANNER_EMULATOR_HOST: localhost:9010
        WEBHOOK_SECRET: test-secret
        PORT: 8080
        TEMPLATES_PATH: ../web/templates
        STATIC_PATH: ../web/static
      steps:
        - uses: actions/checkout@v4

        - uses: actions/setup-go@v5
          with:
            go-version-file: apps/api/go.mod

        - name: Run migrations
          run: go run ./cmd/migrate

        - name: Build and start server
          run: |
            go build -o /tmp/ims-server ./cmd/server
            /tmp/ims-server &
            echo $! > /tmp/server.pid
            for i in {1..30}; do
              if curl -sf "http://localhost:8080/api/companies" > /dev/null; then
                echo "Server ready"
                exit 0
              fi
              sleep 1
            done
            echo "Server did not become ready" >&2
            exit 1

        - name: Run e2e tests
          run: go test ./e2e/...

        - name: Stop server
          if: always()
          run: kill $(cat /tmp/server.pid) || true
  ```

- [ ] **Step 2: Verify the YAML parses**

  Run:
  ```bash
  cd /Users/lamka/Documents/Scalica/invisible-ms
  python3 -c "import yaml; yaml.safe_load(open('.github/workflows/pr-tests.yml'))"
  ```

  Expected: no output and exit code 0.

- [ ] **Step 3: Commit**

  ```bash
  git add .github/workflows/pr-tests.yml
  git commit -m "ci: add PR test workflow with unit and e2e jobs"
  ```

---

### Task 2: Create the e2e test package

**Files:**
- Create: `apps/api/e2e/e2e_test.go`

- [ ] **Step 1: Write the e2e test file**

  Create `apps/api/e2e/e2e_test.go` with this exact content:

  ```go
  package e2e

  import (
  	"bytes"
  	"encoding/json"
  	"io"
  	"net/http"
  	"net/url"
  	"os"
  	"strings"
  	"testing"
  	"time"
  )

  const companyCode = "E2E"

  var (
  	baseURL       string
  	webhookSecret string
  )

  func TestMain(m *testing.M) {
  	baseURL = os.Getenv("E2E_BASE_URL")
  	if baseURL == "" {
  		baseURL = "http://localhost:8080"
  	}
  	webhookSecret = os.Getenv("WEBHOOK_SECRET")
  	if webhookSecret == "" {
  		webhookSecret = "test-secret"
  	}
  	os.Exit(m.Run())
  }

  func skipIfNoEmulator(t *testing.T) {
  	t.Helper()
  	if os.Getenv("GCP_SPANNER_EMULATOR_HOST") == "" {
  		t.Skip("GCP_SPANNER_EMULATOR_HOST not set; skipping e2e tests")
  	}
  }

  type companyResponse struct {
  	CompanyCode string                 `json:"company_code"`
  	CompanyName string                 `json:"company_name"`
  	Roles       map[string]roleResponse `json:"roles"`
  }

  type roleResponse struct {
  	Name       string  `json:"name"`
  	HourlyRate float64 `json:"hourly_rate"`
  }

  type staffResponse struct {
  	StaffID       string   `json:"staff_id"`
  	PhoneNumber   string   `json:"phone_number"`
  	Name          string   `json:"name"`
  	CompanyCode   string   `json:"company_code"`
  	AssignedRoles []string `json:"assigned_roles"`
  	IsActive      bool     `json:"is_active"`
  }

  type activityResponse struct {
  	LogID       string    `json:"log_id"`
  	StaffID     string    `json:"staff_id"`
  	CompanyCode string    `json:"company_code"`
  	Role        string    `json:"role"`
  	ActionType  string    `json:"action_type"`
  	Timestamp   time.Time `json:"timestamp"`
  }

  type sessionResponse struct {
  	StaffID     string    `json:"staff_id"`
  	CompanyCode string    `json:"company_code"`
  	Role        string    `json:"role"`
  	CheckIn     time.Time `json:"check_in"`
  	CheckOut    time.Time `json:"check_out"`
  	Duration    float64   `json:"duration_hours"`
  	Cost        float64   `json:"cost"`
  }

  type dashboardStatsResponse struct {
  	TodayOverview struct {
  		TotalHoursToday float64 `json:"total_hours_today"`
  	} `json:"today_overview"`
  	CostTracking struct {
  		TodayCost float64 `json:"today_cost"`
  	} `json:"cost_tracking"`
  }

  func doRequest(t *testing.T, client *http.Client, method, path string, body any) *http.Response {
  	t.Helper()
  	var bodyReader io.Reader
  	if body != nil {
  		b, err := json.Marshal(body)
  		if err != nil {
  			t.Fatalf("marshal request body: %v", err)
  		}
  		bodyReader = bytes.NewReader(b)
  	}
  	req, err := http.NewRequest(method, baseURL+path, bodyReader)
  	if err != nil {
  		t.Fatalf("create request: %v", err)
  	}
  	if body != nil {
  		req.Header.Set("Content-Type", "application/json")
  	}
  	resp, err := client.Do(req)
  	if err != nil {
  		t.Fatalf("do request: %v", err)
  	}
  	return resp
  }

  func requireStatus(t *testing.T, resp *http.Response, want int) {
  	t.Helper()
  	if resp.StatusCode != want {
  		body, _ := io.ReadAll(resp.Body)
  		t.Fatalf("status: got %d, want %d, body: %s", resp.StatusCode, want, body)
  	}
  }

  func createCompany(t *testing.T, client *http.Client, code, name string) companyResponse {
  	t.Helper()
  	resp := doRequest(t, client, http.MethodPost, "/api/companies", map[string]any{
  		"company_code": code,
  		"company_name": name,
  	})
  	defer resp.Body.Close()
  	requireStatus(t, resp, http.StatusCreated)
  	var c companyResponse
  	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
  		t.Fatalf("decode company: %v", err)
  	}
  	if c.CompanyCode != code {
  		t.Fatalf("company code: got %q, want %q", c.CompanyCode, code)
  	}
  	return c
  }

  func getCompany(t *testing.T, client *http.Client, code string) companyResponse {
  	t.Helper()
  	resp := doRequest(t, client, http.MethodGet, "/api/companies/"+code, nil)
  	defer resp.Body.Close()
  	requireStatus(t, resp, http.StatusOK)
  	var c companyResponse
  	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
  		t.Fatalf("decode company: %v", err)
  	}
  	return c
  }

  func listCompanies(t *testing.T, client *http.Client) []companyResponse {
  	t.Helper()
  	resp := doRequest(t, client, http.MethodGet, "/api/companies", nil)
  	defer resp.Body.Close()
  	requireStatus(t, resp, http.StatusOK)
  	var list []companyResponse
  	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
  		t.Fatalf("decode companies: %v", err)
  	}
  	return list
  }

  func addRole(t *testing.T, client *http.Client, code, role string, rate float64) {
  	t.Helper()
  	resp := doRequest(t, client, http.MethodPost, "/api/companies/"+code+"/roles", map[string]any{
  		"role_name":   role,
  		"hourly_rate": rate,
  	})
  	defer resp.Body.Close()
  	requireStatus(t, resp, http.StatusCreated)
  }

  func updateRole(t *testing.T, client *http.Client, code, role string, rate float64) {
  	t.Helper()
  	resp := doRequest(t, client, http.MethodPut, "/api/companies/"+code+"/roles/"+role, map[string]any{
  		"hourly_rate": rate,
  	})
  	defer resp.Body.Close()
  	requireStatus(t, resp, http.StatusNoContent)
  }

  func deleteRole(t *testing.T, client *http.Client, code, role string) {
  	t.Helper()
  	resp := doRequest(t, client, http.MethodDelete, "/api/companies/"+code+"/roles/"+role, nil)
  	defer resp.Body.Close()
  	requireStatus(t, resp, http.StatusNoContent)
  }

  func createActionType(t *testing.T, client *http.Client, code, actionType, keyword string) {
  	t.Helper()
  	resp := doRequest(t, client, http.MethodPost, "/api/companies/"+code+"/action-types", map[string]any{
  		"action_type": actionType,
  		"keyword":     keyword,
  	})
  	defer resp.Body.Close()
  	requireStatus(t, resp, http.StatusCreated)
  }

  func updateActionTypeKeyword(t *testing.T, client *http.Client, code, actionType, keyword string) {
  	t.Helper()
  	resp := doRequest(t, client, http.MethodPut, "/api/companies/"+code+"/action-types/"+actionType, map[string]any{
  		"keyword": keyword,
  	})
  	defer resp.Body.Close()
  	requireStatus(t, resp, http.StatusNoContent)
  }

  func deleteActionType(t *testing.T, client *http.Client, code, actionType string) {
  	t.Helper()
  	resp := doRequest(t, client, http.MethodDelete, "/api/companies/"+code+"/action-types/"+actionType, nil)
  	defer resp.Body.Close()
  	requireStatus(t, resp, http.StatusNoContent)
  }

  func createStaff(t *testing.T, client *http.Client, phone, name, code string, roles []string) staffResponse {
  	t.Helper()
  	resp := doRequest(t, client, http.MethodPost, "/api/staff", map[string]any{
  		"phone_number": phone,
  		"name":         name,
  		"company_code": code,
  		"roles":        roles,
  	})
  	defer resp.Body.Close()
  	requireStatus(t, resp, http.StatusCreated)
  	var s staffResponse
  	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
  		t.Fatalf("decode staff: %v", err)
  	}
  	if s.Name != name {
  		t.Fatalf("staff name: got %q, want %q", s.Name, name)
  	}
  	return s
  }

  func getStaff(t *testing.T, client *http.Client, id string) staffResponse {
  	t.Helper()
  	resp := doRequest(t, client, http.MethodGet, "/api/staff/"+id, nil)
  	defer resp.Body.Close()
  	requireStatus(t, resp, http.StatusOK)
  	var s staffResponse
  	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
  		t.Fatalf("decode staff: %v", err)
  	}
  	return s
  }

  func listStaff(t *testing.T, client *http.Client, code string) []staffResponse {
  	t.Helper()
  	resp := doRequest(t, client, http.MethodGet, "/api/staff?company_code="+url.QueryEscape(code), nil)
  	defer resp.Body.Close()
  	requireStatus(t, resp, http.StatusOK)
  	var list []staffResponse
  	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
  		t.Fatalf("decode staff list: %v", err)
  	}
  	return list
  }

  func assignRole(t *testing.T, client *http.Client, id, role string) {
  	t.Helper()
  	resp := doRequest(t, client, http.MethodPost, "/api/staff/"+id+"/roles", map[string]any{
  		"role_name": role,
  	})
  	defer resp.Body.Close()
  	requireStatus(t, resp, http.StatusCreated)
  }

  func unassignRole(t *testing.T, client *http.Client, id, role string) {
  	t.Helper()
  	resp := doRequest(t, client, http.MethodDelete, "/api/staff/"+id+"/roles/"+role, nil)
  	defer resp.Body.Close()
  	requireStatus(t, resp, http.StatusNoContent)
  }

  func sendWebhook(t *testing.T, client *http.Client, phone, message, code string) activityResponse {
  	t.Helper()
  	reqBody := map[string]any{
  		"phone":        phone,
  		"message":      message,
  		"company_code": code,
  	}
  	b, err := json.Marshal(reqBody)
  	if err != nil {
  		t.Fatalf("marshal webhook body: %v", err)
  	}
  	req, err := http.NewRequest(http.MethodPost, baseURL+"/webhook/message", bytes.NewReader(b))
  	if err != nil {
  		t.Fatalf("create webhook request: %v", err)
  	}
  	req.Header.Set("Content-Type", "application/json")
  	req.Header.Set("X-Webhook-Secret", webhookSecret)
  	resp, err := client.Do(req)
  	if err != nil {
  		t.Fatalf("send webhook: %v", err)
  	}
  	defer resp.Body.Close()
  	requireStatus(t, resp, http.StatusCreated)
  	var log activityResponse
  	if err := json.NewDecoder(resp.Body).Decode(&log); err != nil {
  		t.Fatalf("decode webhook response: %v", err)
  	}
  	return log
  }

  func listActivities(t *testing.T, client *http.Client, code string) []activityResponse {
  	t.Helper()
  	path := "/api/activities?company_code=" + url.QueryEscape(code)
  	resp := doRequest(t, client, http.MethodGet, path, nil)
  	defer resp.Body.Close()
  	requireStatus(t, resp, http.StatusOK)
  	var list []activityResponse
  	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
  		t.Fatalf("decode activities: %v", err)
  	}
  	return list
  }

  func listSessions(t *testing.T, client *http.Client, code string) []sessionResponse {
  	t.Helper()
  	path := "/api/activities/sessions?company_code=" + url.QueryEscape(code)
  	resp := doRequest(t, client, http.MethodGet, path, nil)
  	defer resp.Body.Close()
  	requireStatus(t, resp, http.StatusOK)
  	var list []sessionResponse
  	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
  		t.Fatalf("decode sessions: %v", err)
  	}
  	return list
  }

  func getDashboardStats(t *testing.T, client *http.Client, code string) dashboardStatsResponse {
  	t.Helper()
  	path := "/api/dashboard/stats?company_code=" + url.QueryEscape(code)
  	resp := doRequest(t, client, http.MethodGet, path, nil)
  	defer resp.Body.Close()
  	requireStatus(t, resp, http.StatusOK)
  	var stats dashboardStatsResponse
  	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
  		t.Fatalf("decode dashboard stats: %v", err)
  	}
  	return stats
  }

  func getDashboardPage(t *testing.T, client *http.Client, code string) {
  	t.Helper()
  	path := "/dashboard?company_code=" + url.QueryEscape(code)
  	resp := doRequest(t, client, http.MethodGet, path, nil)
  	defer resp.Body.Close()
  	requireStatus(t, resp, http.StatusOK)
  	ct := resp.Header.Get("Content-Type")
  	if !strings.Contains(ct, "text/html") {
  		t.Fatalf("dashboard content-type: got %q, want text/html", ct)
  	}
  }

  func TestEndToEnd_CRUDCycle(t *testing.T) {
  	skipIfNoEmulator(t)
  	client := &http.Client{Timeout: 10 * time.Second}

  	// Company CRUD
  	createCompany(t, client, companyCode, "E2E Company")
  	company := getCompany(t, client, companyCode)
  	if company.CompanyCode != companyCode {
  		t.Fatalf("company code: got %q, want %q", company.CompanyCode, companyCode)
  	}
  	companies := listCompanies(t, client)
  	found := false
  	for _, c := range companies {
  		if c.CompanyCode == companyCode {
  			found = true
  			break
  		}
  	}
  	if !found {
  		t.Fatal("created company not found in list")
  	}

  	// Roles
  	addRole(t, client, companyCode, "CLEANING", 15.50)
  	updateRole(t, client, companyCode, "CLEANING", 20.00)
  	addRole(t, client, companyCode, "SECURITY", 25.00)

  	// Action types
  	createActionType(t, client, companyCode, "BREAK_START", "BREAK")
  	updateActionTypeKeyword(t, client, companyCode, "BREAK_START", "PAUSE")
  	deleteActionType(t, client, companyCode, "BREAK_START")

  	// Staff
  	staff := createStaff(t, client, "+1234567890", "Alice", companyCode, []string{"CLEANING"})
  	if len(staff.AssignedRoles) != 1 {
  		t.Fatalf("assigned roles count: got %d, want 1", len(staff.AssignedRoles))
  	}
  	staffByID := getStaff(t, client, staff.StaffID)
  	if staffByID.Name != "Alice" {
  		t.Fatalf("staff name: got %q, want Alice", staffByID.Name)
  	}
  	staffList := listStaff(t, client, companyCode)
  	if len(staffList) != 1 {
  		t.Fatalf("staff list length: got %d, want 1", len(staffList))
  	}
  	assignRole(t, client, staff.StaffID, "SECURITY")
  	unassignRole(t, client, staff.StaffID, "SECURITY")

  	// Delete role after it is no longer assigned
  	deleteRole(t, client, companyCode, "SECURITY")

  	// Activity
  	inLog := sendWebhook(t, client, staff.PhoneNumber, "IN", companyCode)
  	if inLog.ActionType != "CHECK_IN" {
  		t.Fatalf("in action type: got %q, want CHECK_IN", inLog.ActionType)
  	}
  	time.Sleep(100 * time.Millisecond)
  	outLog := sendWebhook(t, client, staff.PhoneNumber, "OUT", companyCode)
  	if outLog.ActionType != "CHECK_OUT" {
  		t.Fatalf("out action type: got %q, want CHECK_OUT", outLog.ActionType)
  	}

  	activities := listActivities(t, client, companyCode)
  	if len(activities) != 2 {
  		t.Fatalf("activities count: got %d, want 2", len(activities))
  	}

  	sessions := listSessions(t, client, companyCode)
  	if len(sessions) != 1 {
  		t.Fatalf("sessions count: got %d, want 1", len(sessions))
  	}
  	if sessions[0].Duration <= 0 {
  		t.Fatalf("session duration: got %f, want > 0", sessions[0].Duration)
  	}
  	if sessions[0].Cost <= 0 {
  		t.Fatalf("session cost: got %f, want > 0", sessions[0].Cost)
  	}

  	// Dashboard
  	stats := getDashboardStats(t, client, companyCode)
  	if stats.TodayOverview.TotalHoursToday <= 0 {
  		t.Fatalf("today hours: got %f, want > 0", stats.TodayOverview.TotalHoursToday)
  	}
  	if stats.CostTracking.TodayCost <= 0 {
  		t.Fatalf("today cost: got %f, want > 0", stats.CostTracking.TodayCost)
  	}
  	getDashboardPage(t, client, companyCode)
  }
  ```

- [ ] **Step 2: Check that the test compiles without running it**

  Run:
  ```bash
  cd /Users/lamka/Documents/Scalica/invisible-ms/apps/api
  go test -run=^$ ./e2e/...
  ```

  Expected: `ok` or `no test files` message and exit code 0. Because the emulator host is not set, the test function will be compiled but not run.

- [ ] **Step 3: Commit**

  ```bash
  git add apps/api/e2e/e2e_test.go
  git commit -m "test: add e2e CRUD cycle test"
  ```

---

### Task 3: Verify the e2e test locally

**Files:** none

- [ ] **Step 1: Start the Spanner emulator**

  Run:
  ```bash
  docker run -d --name e2e-spanner -p 9010:9010 gcr.io/cloud-spanner-emulator/emulator
  ```

  Expected: a container ID is printed.

- [ ] **Step 2: Run migrations and start the server**

  In one terminal, run:
  ```bash
  cd /Users/lamka/Documents/Scalica/invisible-ms/apps/api
  export GCP_SPANNER_PROJECT_ID=invisible-ms-local
  export GCP_SPANNER_INSTANCE_ID=invisible-ms-instance
  export GCP_SPANNER_DATABASE_ID=invisible-ms-db
  export GCP_SPANNER_EMULATOR_HOST=localhost:9010
  export WEBHOOK_SECRET=test-secret
  export PORT=8080
  go run ./cmd/migrate
  go run ./cmd/server
  ```

- [ ] **Step 3: Run the e2e tests**

  In a second terminal, run:
  ```bash
  cd /Users/lamka/Documents/Scalica/invisible-ms/apps/api
  export GCP_SPANNER_EMULATOR_HOST=localhost:9010
  export E2E_BASE_URL=http://localhost:8080
  export WEBHOOK_SECRET=test-secret
  go test -v ./e2e/...
  ```

  Expected: `PASS` for `TestEndToEnd_CRUDCycle`.

- [ ] **Step 4: Stop the emulator**

  Run:
  ```bash
  docker stop e2e-spanner && docker rm e2e-spanner
  ```

---

### Task 4: Verify unit tests still pass

**Files:** none

- [ ] **Step 1: Run unit tests without the emulator**

  Run:
  ```bash
  cd /Users/lamka/Documents/Scalica/invisible-ms/apps/api
  go test ./...
  ```

  Expected: all packages pass, e2e tests are skipped because `GCP_SPANNER_EMULATOR_HOST` is not set.

---

### Task 5: Final review and push

**Files:** none

- [ ] **Step 1: Review the commits**

  Run:
  ```bash
  git log --oneline -5
  ```

  Expected to see:
  - `test: add e2e CRUD cycle test`
  - `ci: add PR test workflow with unit and e2e jobs`
  - the earlier design commit

- [ ] **Step 2: Push the branch**

  Run:
  ```bash
  git push origin <branch-name>
  ```

  Then open a pull request to confirm the workflow runs.
