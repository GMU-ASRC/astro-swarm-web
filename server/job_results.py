from datetime import datetime, timezone

import benchmark_hash
import benchmark_store
import merge


def annotate_rates(results, evaluation):
    if not isinstance(results, dict):
        return
    rate = results.get("success_rate")
    if rate is None:
        return
    rate = float(rate)
    if evaluation.is_attack_level():
        results["attacker_rate"] = round(rate, 1)
        results["defender_rate"] = round(100.0 - rate, 1)
    else:
        results["defender_rate"] = round(rate, 1)
        results["attacker_rate"] = round(100.0 - rate, 1)


def link_benchmark(evaluation, results, replays):
    content_hash = benchmark_hash.for_finished_job(evaluation)
    existing = benchmark_store.find_by_hash(content_hash)
    if existing is not None:
        return existing.id
    benchmark = benchmark_store.build_benchmark(
        evaluation.id,
        evaluation.level_id,
        results,
        replays,
        content_hash=content_hash,
        source_evaluation_id=evaluation.id,
    )
    return benchmark.id


def store(evaluation, payload):
    results, replays = merge.summarize(payload)
    annotate_rates(results, evaluation)
    previous_benchmark_id = evaluation.benchmark_id
    evaluation.benchmark_id = link_benchmark(evaluation, results, replays)
    now = datetime.now(timezone.utc)
    evaluation.status = "done"
    evaluation.progress = 1.0
    evaluation.done_units = evaluation.total_units or 1
    evaluation.error = None
    evaluation.completed_at = now
    evaluation.worker_id = None
    evaluation.stage = None
    evaluation.last_update = now
    if previous_benchmark_id and previous_benchmark_id != evaluation.benchmark_id:
        benchmark_store.release(previous_benchmark_id)
