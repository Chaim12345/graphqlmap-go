#!/usr/bin/env python3
"""
Simple script to test a GraphQL endpoint from CI.
Usage: python3 scripts/test_query.py <graphql_url>
"""
import sys
import requests


def main():
    url = sys.argv[1] if len(sys.argv) > 1 else "http://localhost:8080/graphql"
    query = sys.argv[2] if len(sys.argv) > 2 else "{ __typename }"

    resp = requests.post(url, json={"query": query}, timeout=10)
    print(f"Status: {resp.status_code}")
    print(f"Response: {resp.json()}")

    if resp.status_code != 200:
        sys.exit(1)


if __name__ == "__main__":
    main()
