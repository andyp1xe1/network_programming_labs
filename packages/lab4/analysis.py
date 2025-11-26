#!/usr/bin/env python3
import requests
import time
import concurrent.futures
import matplotlib.pyplot as plt
import numpy as np

LEADER_URL = "http://localhost:9000"
FOLLOWER_PORTS = [9001, 9002, 9003, 9004, 9005]

KEYS=11

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


def timed_set(key, value):
    start = time.time()
    response = requests.post(f"{LEADER_URL}/set", json={"key": key, "value": value})
    duration = (time.time() - start) * 1000  # ms
    return duration, response.status_code == 200


def get_value(url, key):
    try:
        response = requests.post(f"{url}/get", json={"key": key}, timeout=1)
        return response.json().get("value", "") if response.status_code == 200 else ""
    except Exception as e:
        print(f"Error getting {key} from {url}: {e}")
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
        futures = [executor.submit(timed_set, key, value) for key, value in tasks]
        for future in concurrent.futures.as_completed(futures):
            duration, success = future.result()
            if success:
                latencies.append(duration)

    avg_latency = np.mean(latencies) if latencies else 0
    print(f"Writes: {len(latencies)}/100, avg latency: {avg_latency:.1f}ms")

    time.sleep(4)  # Replication settlement

    accuracy, consistent, total = check_consistency(keys)
    print(f"Consistency: {consistent}/{total} nodes ({accuracy * 100:.1f}%)")

    return {
        "quorum": quorum,
        "latency": avg_latency,
        "accuracy": accuracy,
        "writes": len(latencies),
    }


def main():
    print("KV Store Analysis")

    results = []
    for quorum in range(1, 6):
        result = run_test(quorum)
        if result:
            results.append(result)
        time.sleep(2)

    # Extract data
    quorums = [r["quorum"] for r in results]
    latencies = [r["latency"] for r in results]
    accuracies = [r["accuracy"] * 100 for r in results]

    # Plot results
    fig, axes = plt.subplots(1, 3, figsize=(15, 5))

    # Accuracy vs Quorum
    axes[0].plot(quorums, accuracies, "bo-", linewidth=2, markersize=8)
    axes[0].set_xlabel("Write Quorum")
    axes[0].set_ylabel("Consistency (%)")
    axes[0].set_title("Consistency vs Quorum")
    axes[0].grid(True, alpha=0.3)
    axes[0].set_ylim(0, 105)

    # Latency vs Quorum
    axes[1].plot(quorums, latencies, "ro-", linewidth=2, markersize=8)
    axes[1].set_xlabel("Write Quorum")
    axes[1].set_ylabel("Latency (ms)")
    axes[1].set_title("Latency vs Quorum")
    axes[1].grid(True, alpha=0.3)

    # Accuracy vs Latency
    axes[2].scatter(latencies, accuracies, c=quorums, s=100, cmap="viridis")
    axes[2].set_xlabel("Latency (ms)")
    axes[2].set_ylabel("Consistency (%)")
    axes[2].set_title("Consistency vs Latency")
    axes[2].grid(True, alpha=0.3)
    for i, q in enumerate(quorums):
        axes[2].annotate(
            f"Q{q}",
            (latencies[i], accuracies[i]),
            xytext=(5, 5),
            textcoords="offset points",
            fontsize=10,
        )

    plt.tight_layout()
    plt.savefig("analysis_results.png", dpi=150, bbox_inches="tight")
    print("Results saved to analysis_results.png")

    # Print summary
    print("\nSummary:")
    for r in results:
        print(
            f"Quorum {r['quorum']}: {r['latency']:.1f}ms latency, {r['accuracy'] * 100:.1f}% consistency"
        )


if __name__ == "__main__":
    main()
