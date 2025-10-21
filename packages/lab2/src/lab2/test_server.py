#!/usr/bin/env python3

import requests
import time
import sys
import argparse
from concurrent.futures import ThreadPoolExecutor
from collections import defaultdict
import threading


class ServerTester:
    def __init__(self, base_url):
        self.base_url = base_url
        self.stats = {
            'successful': 0,
            'rate_limited': 0,
            'errors': 0,
            'total': 0
        }
        self.response_times = []
        self.lock = threading.Lock()
    
    def make_request(self, url, request_id=None):
        """Make a single HTTP request and record stats"""
        start_time = time.time()
        try:
            response = requests.get(url, timeout=5)
            end_time = time.time()
            response_time = end_time - start_time
            
            with self.lock:
                self.response_times.append(response_time)
                self.stats['total'] += 1
                
                if response.status_code == 200:
                    self.stats['successful'] += 1
                elif response.status_code == 429:
                    self.stats['rate_limited'] += 1
                else:
                    self.stats['errors'] += 1
            
            if request_id:
                status = "SUCCESS" if response.status_code == 200 else f"HTTP {response.status_code}"
                print(f"Request {request_id}: {status}, Time: {response_time:.3f}s")
            
            return response_time, response.status_code
            
        except Exception as e:
            end_time = time.time()
            response_time = end_time - start_time
            
            with self.lock:
                self.response_times.append(response_time)
                self.stats['total'] += 1
                self.stats['errors'] += 1
            
            if request_id:
                print(f"Request {request_id}: ERROR {e}, Time: {response_time:.3f}s")
            
            return response_time, 0
    
    def test_concurrent_load(self, num_requests, max_workers=None):
        """Test server with concurrent requests"""
        print(f"\n=== Concurrent Load Test ===")
        print(f"Making {num_requests} concurrent requests to {self.base_url}")
        
        # Reset stats
        self.stats = {'successful': 0, 'rate_limited': 0, 'errors': 0, 'total': 0}
        self.response_times = []
        
        if max_workers is None:
            max_workers = min(num_requests, 20)
        
        start_time = time.time()
        
        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            futures = []
            for i in range(num_requests):
                future = executor.submit(self.make_request, self.base_url, i + 1)
                futures.append(future)
            
            # Wait for all requests to complete
            results = [future.result() for future in futures]
        
        end_time = time.time()
        total_time = end_time - start_time
        
        self._print_results(total_time)
        return total_time, self.stats.copy()
    
    def test_rate_limit(self, requests_per_second, duration_seconds):
        """Test rate limiting by making requests at specified rate"""
        print(f"\n=== Rate Limiting Test ===")
        print(f"Making {requests_per_second} requests/second for {duration_seconds} seconds")
        
        # Reset stats
        self.stats = {'successful': 0, 'rate_limited': 0, 'errors': 0, 'total': 0}
        self.response_times = []
        
        total_requests = requests_per_second * duration_seconds
        interval = 1.0 / requests_per_second
        
        start_time = time.time()
        
        for i in range(total_requests):
            request_start = time.time()
            self.make_request(self.base_url, i + 1)
            
            # Calculate how long to sleep to maintain the rate
            elapsed = time.time() - request_start
            sleep_time = max(0, interval - elapsed)
            if sleep_time > 0:
                time.sleep(sleep_time)
        
        end_time = time.time()
        total_time = end_time - start_time
        
        self._print_results(total_time)
        return total_time, self.stats.copy()
    
    def test_sustained_load(self, requests_per_second, duration_seconds, max_workers=10):
        """Test sustained load using thread pool to maintain request rate"""
        print(f"\n=== Sustained Load Test ===")
        print(f"Maintaining {requests_per_second} requests/second for {duration_seconds} seconds")
        
        # Reset stats
        self.stats = {'successful': 0, 'rate_limited': 0, 'errors': 0, 'total': 0}
        self.response_times = []
        
        interval = 1.0 / requests_per_second
        total_requests = requests_per_second * duration_seconds
        
        start_time = time.time()
        
        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            for i in range(total_requests):
                executor.submit(self.make_request, self.base_url, i + 1)
                time.sleep(interval)
        
        end_time = time.time()
        total_time = end_time - start_time
        
        self._print_results(total_time)
        return total_time, self.stats.copy()
    
    def _print_results(self, total_time):
        """Print comprehensive test results"""
        print(f"\n=== Test Results ===")
        print(f"Total time: {total_time:.2f}s")
        print(f"Total requests: {self.stats['total']}")
        print(f"Successful requests: {self.stats['successful']}")
        print(f"Rate limited (429): {self.stats['rate_limited']}")
        print(f"Other errors: {self.stats['errors']}")
        
        if self.response_times:
            avg_response_time = sum(self.response_times) / len(self.response_times)
            min_response_time = min(self.response_times)
            max_response_time = max(self.response_times)
            
            print(f"\n=== Response Time Stats ===")
            print(f"Average response time: {avg_response_time:.3f}s")
            print(f"Min response time: {min_response_time:.3f}s")
            print(f"Max response time: {max_response_time:.3f}s")
        
        if self.stats['total'] > 0:
            success_rate = (self.stats['successful'] / self.stats['total']) * 100
            throughput = self.stats['successful'] / total_time if total_time > 0 else 0
            
            print(f"\n=== Throughput Stats ===")
            print(f"Success rate: {success_rate:.1f}%")
            print(f"Successful throughput: {throughput:.2f} requests/second")
            print(f"Total throughput: {self.stats['total'] / total_time:.2f} requests/second")


def main():
    parser = argparse.ArgumentParser(description="Test HTTP server performance and rate limiting")
    parser.add_argument("url", help="Server URL to test (e.g., http://localhost:8080/index.html)")
    parser.add_argument("--rps", type=int, default=10, help="Requests per second for rate limiting test (default: 10)")
    parser.add_argument("--duration", type=int, default=5, help="Duration in seconds for rate limiting test (default: 5)")
    parser.add_argument("--concurrent", type=int, default=20, help="Number of concurrent requests for load test (default: 20)")
    parser.add_argument("--workers", type=int, help="Max worker threads (default: auto)")
    parser.add_argument("--test", choices=['load', 'rate', 'sustained', 'all'], default='all', 
                        help="Type of test to run (default: all)")
    
    args = parser.parse_args()
    
    tester = ServerTester(args.url)
    
    if args.test in ['load', 'all']:
        tester.test_concurrent_load(args.concurrent, args.workers)
    
    if args.test in ['rate', 'all']:
        tester.test_rate_limit(args.rps, args.duration)
    
    if args.test in ['sustained', 'all']:
        tester.test_sustained_load(args.rps, args.duration, args.workers or 10)


if __name__ == "__main__":
    main()