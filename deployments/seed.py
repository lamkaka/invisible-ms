#!/usr/bin/env python3
"""Seed IMS with random demo data through the public REST API and webhook."""

import argparse
import os
import random
import string
import time
import uuid

import requests


def env(key, default=None, cast=str):
    value = os.environ.get(key, default)
    if value is None:
        return None
    return cast(value)


FIRST_NAMES = [
    "Emma", "Liam", "Olivia", "Noah", "Ava", "Ethan", "Sophia", "Mason",
    "Isabella", "William", "Mia", "James", "Charlotte", "Benjamin", "Amelia",
    "Lucas", "Harper", "Henry", "Evelyn", "Alexander", "Abigail", "Daniel",
    "Emily", "Matthew", "Ella", "Jackson", "Elizabeth", "Sebastian", "Camila",
    "Jack", "Luna", "Owen", "Sofia", "Samuel", "Avery", "Jacob", "Mila",
    "Aiden", "Aria", "John", "Scarlett", "Joseph", "Penelope", "David",
    "Layla", "Michael", "Chloe", "Carter", "Victoria", "Wyatt"
]
LAST_NAMES = [
    "Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller",
    "Davis", "Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez",
    "Wilson", "Anderson", "Thomas", "Taylor", "Moore", "Jackson", "Martin",
    "Lee", "Perez", "Thompson", "White", "Harris", "Sanchez", "Clark",
    "Ramirez", "Lewis", "Robinson", "Walker", "Young", "Allen", "King",
    "Wright", "Scott", "Torres", "Nguyen", "Hill", "Flores", "Green",
    "Adams", "Nelson", "Baker", "Hall", "Rivera", "Campbell", "Mitchell"
]
ROLE_NAMES = [
    "CLEANING", "SECURITY", "KITCHEN", "RECEPTION", "WAREHOUSE", "DELIVERY",
    "CASHIER", "STOCKER", "BARISTA", "HOSTING", "BARTENDING", "COOKING"
]
COMPANY_ADJECTIVES = [
    "Swift", "Bright", "Golden", "Prime", "Nova", "Blue", "Green", "Red",
    "Urban", "Royal", "Peak", "True", "Smart", "Safe", "Fresh"
]
COMPANY_NOUNS = [
    "Services", "Solutions", "Group", "Logistics", "Hospitality", "Retail",
    "Staffing", "Facilities", "Catering", "Security", "Cleaners", "Care"
]


class SeedClient:
    def __init__(self, base_url, webhook_secret):
        self.base_url = base_url.rstrip("/")
        self.webhook_secret = webhook_secret
        self.session = requests.Session()

    def create_company(self, code, name):
        resp = self.session.post(
            f"{self.base_url}/api/companies",
            json={"company_code": code, "company_name": name},
        )
        resp.raise_for_status()
        return resp.json()

    def create_role(self, company_code, role_name, hourly_rate):
        resp = self.session.post(
            f"{self.base_url}/api/companies/{company_code}/roles",
            json={"role_name": role_name, "hourly_rate": hourly_rate},
        )
        resp.raise_for_status()

    def create_action_type(self, company_code, action_type, keyword):
        resp = self.session.post(
            f"{self.base_url}/api/companies/{company_code}/action-types",
            json={"action_type": action_type, "keyword": keyword},
        )
        resp.raise_for_status()

    def create_staff(self, staff_id, phone, name, company_code, roles):
        resp = self.session.post(
            f"{self.base_url}/api/staff",
            json={
                "staff_id": staff_id,
                "phone_number": phone,
                "name": name,
                "company_code": company_code,
                "roles": roles,
            },
        )
        resp.raise_for_status()
        return resp.json()

    def send_webhook(self, phone, message, company_code):
        resp = self.session.post(
            f"{self.base_url}/webhook/message",
            json={"phone": phone, "message": message, "company_code": company_code},
            headers={"X-Webhook-Secret": self.webhook_secret},
        )
        resp.raise_for_status()
        return resp.json()


def random_phone():
    return f"+1{random.randint(200, 999)}{random.randint(1000000, 9999999)}"


def random_name():
    return f"{random.choice(FIRST_NAMES)} {random.choice(LAST_NAMES)}"


def generate_companies(count, fixed_code=None, fixed_name=None):
    companies = []
    for i in range(count):
        if fixed_code and i == 0:
            companies.append({"code": fixed_code, "name": fixed_name or fixed_code})
        else:
            adj = random.choice(COMPANY_ADJECTIVES)
            noun = random.choice(COMPANY_NOUNS)
            code = f"{adj.upper()}{noun.upper()}"[:15]
            companies.append({"code": code, "name": f"{adj} {noun}"})
    return companies


def generate_roles(company_code, count):
    roles = []
    for role_name in random.sample(ROLE_NAMES, min(count, len(ROLE_NAMES))):
        hourly_rate = round(random.uniform(12.0, 45.0), 2)
        roles.append({
            "company_code": company_code,
            "role_name": role_name,
            "hourly_rate": hourly_rate,
        })
    return roles


def generate_staff(company_code, roles, count):
    staff_list = []
    for _ in range(count):
        staff_id = str(uuid.uuid4())
        phone = random_phone()
        name = random_name()
        assigned_roles = [random.choice(roles)["role_name"]]
        staff_list.append({
            "staff_id": staff_id,
            "phone": phone,
            "name": name,
            "company_code": company_code,
            "roles": assigned_roles,
        })
    return staff_list


def seed(args):
    client = SeedClient(args.api_url, args.webhook_secret)

    companies = generate_companies(args.companies, args.company_code, args.company_name)
    total_staff = 0
    total_checked_in = 0
    total_checked_out = 0

    for company in companies:
        try:
            client.create_company(company["code"], company["name"])
            print(f"Created company {company['code']} - {company['name']}")
        except requests.HTTPError as exc:
            if exc.response.status_code == 409:
                print(f"Company {company['code']} already exists, using it")
            else:
                print(f"Skipping company {company['code']}: {exc.response.text}")
                continue

        # Seed action types (overrides are skipped with 409)
        action_types = [
            ("BREAK_START", "BREAK"),
        ]
        for action_type, keyword in action_types:
            try:
                client.create_action_type(company["code"], action_type, keyword)
                print(f"  Created action type {action_type} ({keyword})")
            except requests.HTTPError as exc:
                if exc.response.status_code == 409:
                    print(f"  Action type {action_type} already exists, skipping")
                else:
                    print(f"  Skipping action type {action_type}: {exc.response.text}")

        roles = generate_roles(company["code"], args.roles_per_company)
        for role in roles:
            try:
                client.create_role(company["code"], role["role_name"], role["hourly_rate"])
                print(f"  Created role {role['role_name']} @ ${role['hourly_rate']}/hr")
            except requests.HTTPError as exc:
                print(f"  Skipping role {role['role_name']}: {exc.response.text}")

        staff_list = generate_staff(company["code"], roles, args.staff_per_company)
        for staff in staff_list:
            try:
                client.create_staff(
                    staff["staff_id"],
                    staff["phone"],
                    staff["name"],
                    staff["company_code"],
                    staff["roles"],
                )
                total_staff += 1
                print(f"    Created staff {staff['name']} ({staff['phone']})")
            except requests.HTTPError as exc:
                print(f"    Skipping staff {staff['phone']}: {exc.response.text}")
                continue

            try:
                client.send_webhook(staff["phone"], "IN", company["code"])
                total_checked_in += 1
                print(f"      -> checked IN")
            except requests.HTTPError as exc:
                print(f"      -> check-in failed: {exc.response.text}")
                continue

            if random.random() < args.checkout_fraction:
                sleep_seconds = random.uniform(0.5, args.max_session_seconds)
                time.sleep(sleep_seconds)
                try:
                    client.send_webhook(staff["phone"], "OUT", company["code"])
                    total_checked_out += 1
                    print(f"      -> checked OUT after {sleep_seconds:.1f}s")
                except requests.HTTPError as exc:
                    print(f"      -> check-out failed: {exc.response.text}")

    print("\n=== Seed Summary ===")
    print(f"Companies created/attempted: {len(companies)}")
    print(f"Staff created: {total_staff}")
    print(f"Checked in: {total_checked_in}")
    print(f"Checked out: {total_checked_out}")
    print(f"Currently working: {total_checked_in - total_checked_out}")


def main():
    parser = argparse.ArgumentParser(description="Seed IMS with random demo data")
    parser.add_argument("--api-url", default=env("API_URL", "http://localhost:8888"))
    parser.add_argument("--webhook-secret", default=env("WEBHOOK_SECRET", "test-secret"))
    parser.add_argument("--companies", type=int, default=env("COMPANIES", "3", int))
    parser.add_argument("--roles-per-company", type=int, default=env("ROLES_PER_COMPANY", "3", int))
    parser.add_argument("--staff-per-company", type=int, default=env("STAFF_PER_COMPANY", "5", int))
    parser.add_argument("--company-code", default=env("COMPANY_CODE", None))
    parser.add_argument("--company-name", default=env("COMPANY_NAME", None))
    parser.add_argument("--checkout-fraction", type=float, default=env("CHECKOUT_FRACTION", "0.3", float))
    parser.add_argument("--max-session-seconds", type=float, default=env("MAX_SESSION_SECONDS", "3.0", float))
    args = parser.parse_args()
    seed(args)


if __name__ == "__main__":
    main()
