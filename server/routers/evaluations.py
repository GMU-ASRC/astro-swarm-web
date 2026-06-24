from collections import defaultdict

from flask import Blueprint, current_app, jsonify, request
from werkzeug.exceptions import BadRequest, Unauthorized

from config import Config
from database import db
from evaluator import run_evaluation_async
from models import PlayerEvaluation
from schemas import EvaluationSubmit

evaluations_bp = Blueprint("evaluations", __name__, url_prefix="/api/evaluations")


@evaluations_bp.get("")
def list_players():
    evaluations = (
        PlayerEvaluation.query.order_by(PlayerEvaluation.created_at.desc()).all()
    )
    latest_by_player = {}
    for evaluation in evaluations:
        if evaluation.player_id not in latest_by_player:
            latest_by_player[evaluation.player_id] = evaluation
    return jsonify([item.to_list_dict() for item in latest_by_player.values()])


@evaluations_bp.post("")
def submit_evaluation():
    if request.headers.get("X-API-Key") != Config.API_SECRET_KEY:
        raise Unauthorized("Invalid API key")

    data = request.get_json(silent=True)
    if not data:
        raise BadRequest("Invalid JSON data")

    try:
        parsed = EvaluationSubmit(**data)
    except Exception as exc:
        raise BadRequest(str(exc))

    evaluation = PlayerEvaluation(
        player_id=parsed.player_id,
        username=parsed.username,
        algorithm=parsed.algorithm,
        n_max=parsed.n_max,
        trials=parsed.trials,
        status="queued",
    )
    db.session.add(evaluation)
    db.session.commit()

    run_evaluation_async(current_app._get_current_object(), evaluation.id)

    return jsonify(evaluation.to_dict()), 202


@evaluations_bp.get("/baseline")
def baseline():
    evaluations = PlayerEvaluation.query.filter_by(status="done").all()
    totals = defaultdict(float)
    counts = defaultdict(int)
    for evaluation in evaluations:
        for point in evaluation.results or []:
            n = point.get("n")
            rate = point.get("success_rate")
            if n is None or rate is None:
                continue
            totals[n] += rate
            counts[n] += 1
    results = [
        {"n": n, "success_rate": round(totals[n] / counts[n], 1)}
        for n in sorted(totals)
    ]
    return jsonify({"results": results, "samples": len(evaluations)})


@evaluations_bp.get("/<player_id>")
def get_player(player_id: str):
    evaluation = (
        PlayerEvaluation.query.filter_by(player_id=player_id)
        .order_by(PlayerEvaluation.created_at.desc())
        .first()
    )
    if evaluation is None:
        raise BadRequest("No evaluation found for this player")
    return jsonify(evaluation.to_dict())
