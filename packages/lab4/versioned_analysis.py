#!/usr/bin/env python3
import requests
import time
import concurrent.futures
import matplotlib.pyplot as plt
import numpy as np

LEADER_URL = "http://localhost:9000"
FOLLOWER_PORTS = [9001, 9002, 9003, 9004, 9005]

KEYS = 10


def set_quorum(quorum):
    try:
        response = requests.post(
            f"{LEADER_URL}/admin/quorum", json={"quorum": quorum}, timeout=5
        )
        print(f"Set quorum {quorum}: {response.status_code}")
        return response.status_code == 200
    except Exception as e:
        print(f"Failed to set quorum {quorum}: {e}")
        return False


def get_with_metadata(url, key):
    """Get a key with its metadata (version, timestamps)"""
    try:
        response = requests.post(f"{url}/get", json={"key": key}, timeout=1)
        if response.status_code == 200:
            data = response.json()
            metadata = data.get("metadata", {})
            return {
                "value": data.get("value", ""),
                "version": metadata.get("version", 0),
            }
        # Key not found is a valid state - return empty result
        elif response.status_code == 500 and "key not found" in response.text:
            return {
                "value": "",
                "version": 0,
            }
        return None
    except Exception:
        return None


def timed_versioned_set(key, value):
    """Timed set with version conflict retry logic using optimistic locking"""
    start = time.time()
    max_retries = 50

    for attempt in range(max_retries):
        # Get current version for optimistic locking
        current_data = get_with_metadata(LEADER_URL, key)
        expected_version = 0  # Default for new keys

        if (
            current_data
            and current_data.get("value")
            and current_data.get("version", 0) > 0
        ):
            # Key exists with a value, use its current version
            expected_version = current_data["version"]

        # Set with the expected version and required id field
        payload = {
            "key": key,
            "value": value,
            "expected_version": expected_version,
        }

        try:
            response = requests.post(f"{LEADER_URL}/set", json=payload, timeout=5)

            # Success case
            if response.status_code == 200:
                duration = (time.time() - start) * 1000  # ms
                return duration, True

            # Version conflict - retry with fresh version
            elif response.status_code == 409:
                if attempt == max_retries - 1:  # Last attempt failed
                    break
                continue  # Retry with fresh version

            # Other error - give up
            else:
                break

        except Exception as e:
            print(f"Error setting {key}: {e}")
            break

    # All attempts failed
    duration = (time.time() - start) * 1000  # ms
    return duration, False


def get_value(url, key):
    try:
        response = requests.post(f"{url}/get", json={"key": key}, timeout=2)
        return response.json().get("value", "") if response.status_code == 200 else ""
    except Exception:
        # Don't print errors during consistency checks to avoid noise
        return ""


def check_consistency(keys):
    leader_data = {key: get_value(LEADER_URL, key) for key in keys}
    leader_data = {k: v for k, v in leader_data.items() if v}

    consistent_followers = 0
    for port in FOLLOWER_PORTS:
        follower_url = f"http://localhost:{port}"
        is_consistent = all(
            get_value(follower_url, key) == value for key, value in leader_data.items()
        )
        if is_consistent:
            consistent_followers += 1

    return (
        consistent_followers / len(FOLLOWER_PORTS),
        consistent_followers,
        len(FOLLOWER_PORTS),
    )


def run_test(quorum):
    print(f"Testing quorum {quorum}...")

    if not set_quorum(quorum):
        return None

    time.sleep(1)  # Config propagation

    # 100 writes on random keys
    keys = [f"key_{i}" for i in range(KEYS)]
    tasks = [(keys[i % KEYS], f"value_{i}_{quorum}") for i in range(100)]

    latencies = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=10) as executor:
        futures = [
            executor.submit(timed_versioned_set, key, value) for key, value in tasks
        ]
        for future in concurrent.futures.as_completed(futures):
            duration, success = future.result()
            if success:
                latencies.append(duration)

    # Calculate latency statistics
    if latencies:
        latency_stats = {
            "mean": np.mean(latencies),
            "median": np.median(latencies),
            "p95": np.percentile(latencies, 95),
            "p99": np.percentile(latencies, 99),
            "min": np.min(latencies),
            "max": np.max(latencies),
        }
    else:
        latency_stats = {
            stat: 0 for stat in ["mean", "median", "p95", "p99", "min", "max"]
        }

    print(f"Writes: {len(latencies)}/100")
    print(
        f"  Mean: {latency_stats['mean']:.1f}ms, Median: {latency_stats['median']:.1f}ms"
    )
    print(f"  P95: {latency_stats['p95']:.1f}ms, P99: {latency_stats['p99']:.1f}ms")

    time.sleep(4)  # Replication settlement

    accuracy, consistent, total = check_consistency(keys)
    print(f"Consistency: {consistent}/{total} nodes ({accuracy * 100:.1f}%)")

    return {
        "quorum": quorum,
        "latency_stats": latency_stats,
        "accuracy": accuracy,
        "writes": len(latencies),
        "raw_latencies": latencies,  # Keep raw data for analysis
    }


def main():
    print("KV Store Versioned Analysis (Optimistic Locking)")

    results = []
    for quorum in range(1, 6):
        result = run_test(quorum)
        if result:
            results.append(result)
        time.sleep(2)

    # Extract data
    quorums = [r["quorum"] for r in results]
    accuracies = [r["accuracy"] * 100 for r in results]

    # Extract latency statistics
    mean_latencies = [r["latency_stats"]["mean"] for r in results]
    median_latencies = [r["latency_stats"]["median"] for r in results]
    p95_latencies = [r["latency_stats"]["p95"] for r in results]
    p99_latencies = [r["latency_stats"]["p99"] for r in results]

    # Plot results
    fig, axes = plt.subplots(1, 3, figsize=(18, 5))

    # Accuracy vs Quorum
    axes[0].plot(quorums, accuracies, "bo-", linewidth=2, markersize=8)
    axes[0].set_xlabel("Write Quorum")
    axes[0].set_ylabel("Consistency (%)")
    axes[0].set_title("Versioned Consistency vs Quorum")
    axes[0].grid(True, alpha=0.3)
    axes[0].set_ylim(0, 105)

    # Latency vs Quorum - Multiple metrics overlapped
    axes[1].plot(
        quorums,
        mean_latencies,
        "ro-",
        linewidth=2,
        markersize=6,
        label="Mean",
        alpha=0.8,
    )
    axes[1].plot(
        quorums,
        median_latencies,
        "go-",
        linewidth=2,
        markersize=6,
        label="Median",
        alpha=0.8,
    )
    axes[1].plot(
        quorums, p95_latencies, "mo-", linewidth=2, markersize=6, label="P95", alpha=0.8
    )
    axes[1].plot(
        quorums, p99_latencies, "co-", linewidth=2, markersize=6, label="P99", alpha=0.8
    )
    axes[1].set_xlabel("Write Quorum")
    axes[1].set_ylabel("Latency (ms)")
    axes[1].set_title("Versioned Latency Metrics vs Quorum")
    axes[1].grid(True, alpha=0.3)
    axes[1].legend()

    # Accuracy vs Latency (using mean latency)
    axes[2].scatter(mean_latencies, accuracies, c=quorums, s=100, cmap="viridis")
    axes[2].set_xlabel("Mean Latency (ms)")
    axes[2].set_ylabel("Consistency (%)")
    axes[2].set_title("Versioned Consistency vs Mean Latency")
    axes[2].grid(True, alpha=0.3)
    for i, q in enumerate(quorums):
        axes[2].annotate(
            f"Q{q}",
            (mean_latencies[i], accuracies[i]),
            xytext=(5, 5),
            textcoords="offset points",
            fontsize=10,
        )

    plt.tight_layout()
    plt.savefig("versioned_analysis_results_fixed.png", dpi=150, bbox_inches="tight")
    print("Results saved to versioned_analysis_results_fixed.png")

    # Print summary
    print("\nVersioned Analysis Summary (with Optimistic Locking):")
    for r in results:
        stats = r["latency_stats"]
        print(
            f"Quorum {r['quorum']}: Mean {stats['mean']:.1f}ms, Median {stats['median']:.1f}ms, "
            f"P95 {stats['p95']:.1f}ms, P99 {stats['p99']:.1f}ms, {r['accuracy'] * 100:.1f}% consistency"
        )


if __name__ == "__main__":
    main()
