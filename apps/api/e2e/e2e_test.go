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

type actionTypeResponse struct {
	ActionType string `json:"action_type"`
	Keyword    string `json:"keyword"`
	IsSystem   bool   `json:"is_system"`
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

func listCompanyRoles(t *testing.T, client *http.Client, code string) []roleResponse {
	t.Helper()
	resp := doRequest(t, client, http.MethodGet, "/api/companies/"+code+"/roles", nil)
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusOK)
	var list []roleResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatalf("decode roles: %v", err)
	}
	return list
}

func listCompanyActionTypes(t *testing.T, client *http.Client, code string) []actionTypeResponse {
	t.Helper()
	resp := doRequest(t, client, http.MethodGet, "/api/companies/"+code+"/action-types", nil)
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusOK)
	var list []actionTypeResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatalf("decode action types: %v", err)
	}
	return list
}

func updateStaff(t *testing.T, client *http.Client, id, name string, roles []string, isActive bool) staffResponse {
	t.Helper()
	resp := doRequest(t, client, http.MethodPut, "/api/staff/"+id, map[string]any{
		"name":           name,
		"assigned_roles": roles,
		"is_active":      isActive,
	})
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusOK)
	var s staffResponse
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		t.Fatalf("decode staff update: %v", err)
	}
	return s
}

func getStaffPage(t *testing.T, client *http.Client) {
	t.Helper()
	resp := doRequest(t, client, http.MethodGet, "/staff", nil)
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusOK)
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("staff page content-type: got %q, want text/html", ct)
	}
}

func getActionsPage(t *testing.T, client *http.Client) {
	t.Helper()
	resp := doRequest(t, client, http.MethodGet, "/actions", nil)
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusOK)
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("actions page content-type: got %q, want text/html", ct)
	}
}

func getRolesPage(t *testing.T, client *http.Client) {
	t.Helper()
	resp := doRequest(t, client, http.MethodGet, "/roles", nil)
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusOK)
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("roles page content-type: got %q, want text/html", ct)
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

	// List roles and verify
	rolesList := listCompanyRoles(t, client, companyCode)
	foundCleaning := false
	foundSecurity := false
	for _, r := range rolesList {
		if r.Name == "CLEANING" {
			foundCleaning = true
			if r.HourlyRate != 20.00 {
				t.Fatalf("CLEANING rate: got %f, want 20.00", r.HourlyRate)
			}
		}
		if r.Name == "SECURITY" {
			foundSecurity = true
			if r.HourlyRate != 25.00 {
				t.Fatalf("SECURITY rate: got %f, want 25.00", r.HourlyRate)
			}
		}
	}
	if !foundCleaning {
		t.Fatal("CLEANING role not found in list")
	}
	if !foundSecurity {
		t.Fatal("SECURITY role not found in list")
	}

	// Action types
	createActionType(t, client, companyCode, "BREAK_START", "BREAK")
	updateActionTypeKeyword(t, client, companyCode, "BREAK_START", "PAUSE")
	deleteActionType(t, client, companyCode, "BREAK_START")

	// List action types and verify system types are present
	actionTypes := listCompanyActionTypes(t, client, companyCode)
	foundCheckIn := false
	foundCheckOut := false
	for _, at := range actionTypes {
		if at.ActionType == "CHECK_IN" {
			foundCheckIn = true
			if !at.IsSystem {
				t.Fatal("CHECK_IN should be a system action type")
			}
		}
		if at.ActionType == "CHECK_OUT" {
			foundCheckOut = true
			if !at.IsSystem {
				t.Fatal("CHECK_OUT should be a system action type")
			}
		}
		if at.ActionType == "BREAK_START" {
			t.Fatal("BREAK_START should have been deleted")
		}
	}
	if !foundCheckIn {
		t.Fatal("CHECK_IN action type not found")
	}
	if !foundCheckOut {
		t.Fatal("CHECK_OUT action type not found")
	}

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

	// Update staff
	updatedStaff := updateStaff(t, client, staff.StaffID, "Alice Updated", []string{"CLEANING"}, true)
	if updatedStaff.Name != "Alice Updated" {
		t.Fatalf("updated staff name: got %q, want Alice Updated", updatedStaff.Name)
	}
	staffByID = getStaff(t, client, staff.StaffID)
	if staffByID.Name != "Alice Updated" {
		t.Fatalf("retrieved staff name after update: got %q, want Alice Updated", staffByID.Name)
	}

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

	// HTML page endpoints
	getStaffPage(t, client)
	getActionsPage(t, client)
	getRolesPage(t, client)
}
