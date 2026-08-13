#!/usr/bin/env python3
"""
Demo GraphQL server for testing both Go and Python implementations.
Runs in CI as a service container.
"""

import json
import time
from http.server import HTTPServer, BaseHTTPRequestHandler
import os

class GraphQLHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        if self.path == '/graphql':
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            
            try:
                request = json.loads(post_data.decode('utf-8'))
                query = request.get('query', '')
                response = self.execute_query(query)
                
                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(json.dumps(response).encode())
            except json.JSONDecodeError:
                self.send_error_response("Invalid JSON")
        else:
            self.send_error(404)
    
    def do_GET(self):
        if self.path == '/health':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps({"status": "ok"}).encode())
        else:
            self.send_error(404)
    
    def execute_query(self, query):
        query = query.strip()
        
        # Handle introspection
        if '__schema' in query:
            return {
                "data": {
                    "__schema": {
                        "queryType": {"name": "Query"},
                        "mutationType": {"name": "Mutation"},
                        "types": [
                            {"name": "Query", "kind": "OBJECT"},
                            {"name": "Mutation", "kind": "OBJECT"},
                            {"name": "User", "kind": "OBJECT"},
                            {"name": "String", "kind": "SCALAR"},
                            {"name": "Int", "kind": "SCALAR"}
                        ]
                    }
                }
            }
        
        # Handle __typename
        if '__typename' in query:
            return {"data": {"__typename": "Query"}}
        
        # Handle user queries
        if 'user' in query:
            if 'test1' in query or 'test3' in query:
                return {
                    "data": {
                        "user": {"id": "123", "name": "Interesting User"}
                    }
                }
            return {
                "data": {
                    "user": {"id": "456", "name": "Test User"}
                }
            }
        
        # Handle search queries
        if 'search' in query:
            return {
                "data": {
                    "search": [
                        {"id": "1", "name": "Result 1"},
                        {"id": "2", "name": "Result 2"}
                    ]
                }
            }
        
        # Handle injection tests
        if 'DROP TABLE' in query or "OR '1'='1" in query:
            return {
                "errors": [
                    {"message": "Syntax Error: Unexpected Name", "locations": [{"line": 1, "column": 2}]}
                ]
            }
        
        # Handle blind injection with delay
        if 'sleep' in query or 'WAITFOR' in query:
            time.sleep(6)
            return {"data": {"result": "delayed"}}
        
        # Default response
        return {"data": {"result": "ok"}}
    
    def send_error_response(self, message):
        self.send_response(400)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        response = {"errors": [{"message": message}]}
        self.wfile.write(json.dumps(response).encode())
    
    def log_message(self, format, *args):
        # Suppress default logging
        pass

def main():
    port = int(os.environ.get('PORT', 8080))
    server = HTTPServer(('0.0.0.0', port), GraphQLHandler)
    print(f"Demo GraphQL server starting on port {port}")
    server.serve_forever()

if __name__ == '__main__':
    main()
