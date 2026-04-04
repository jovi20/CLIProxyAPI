DEFAULT_MODEL = "gpt-3.5-turbo-0125"

GPT5_MODEL_ALIASES = {
    "gpt-5.4": "gpt-5.4",
    "gpt-5.2": "gpt-5.4",
    "gpt-5.2-codex": "gpt-5.4",
    "gpt-5.3-codex": "gpt-5.4",
    "gpt-5.1-codex-max": "gpt-5.4",
    "gpt-5.4-mini": "gpt-5.4-mini",
    "gpt-5": "gpt-5.3",
    "gpt-5-codex": "gpt-5.3",
    "gpt-5-codex-mini": "gpt-5.3",
    "gpt-5.1": "gpt-5.3",
    "gpt-5.1-codex": "gpt-5.3",
    "gpt-5.1-codex-mini": "gpt-5.3",
    "gpt-5.3": "gpt-5.3",
    "gpt-5.3-codex-spark": "gpt-5.3",
}


def split_model_name_and_suffix(raw_model):
    if not isinstance(raw_model, str):
        return DEFAULT_MODEL, ""

    trimmed = raw_model.strip()
    if not trimmed:
        return DEFAULT_MODEL, ""

    if not trimmed.endswith(")") or "(" not in trimmed:
        return trimmed, ""

    base, _, suffix = trimmed.rpartition("(")
    base = base.strip()
    suffix = suffix[:-1].strip()
    if not base or not suffix:
        return trimmed, ""
    return base, suffix


def normalize_reasoning_effort(value):
    if not isinstance(value, str):
        return None
    effort = value.strip().lower()
    if not effort:
        return None
    return effort


def extract_reasoning_effort(payload, model_suffix=""):
    if isinstance(payload, dict):
        effort = normalize_reasoning_effort(payload.get("reasoning_effort"))
        if effort:
            return effort

        reasoning = payload.get("reasoning")
        if isinstance(reasoning, dict):
            effort = normalize_reasoning_effort(reasoning.get("effort"))
            if effort:
                return effort

    return normalize_reasoning_effort(model_suffix)


def resolve_request_model(origin_model):
    if not isinstance(origin_model, str):
        return "gpt-4o"

    model = origin_model.strip()
    if not model:
        return "gpt-4o"

    lowered = model.lower()
    if lowered in GPT5_MODEL_ALIASES:
        return GPT5_MODEL_ALIASES[lowered]
    if "o3-mini-high" in lowered:
        return "o3-mini-high"
    if "o3-mini-medium" in lowered:
        return "o3-mini-medium"
    if "o3-mini-low" in lowered:
        return "o3-mini-low"
    if "o3-mini" in lowered:
        return "o3-mini"
    if "o3" in lowered:
        return "o3"
    if "o1-preview" in lowered:
        return "o1-preview"
    if "o1-pro" in lowered:
        return "o1-pro"
    if "o1-mini" in lowered:
        return "o1-mini"
    if "o1" in lowered:
        return "o1"
    if "gpt-4.5o" in lowered:
        return "gpt-4.5o"
    if "gpt-4o-canmore" in lowered:
        return "gpt-4o-canmore"
    if "gpt-4o-mini" in lowered:
        return "gpt-4o-mini"
    if "gpt-4o" in lowered:
        return "gpt-4o"
    if "gpt-4-mobile" in lowered:
        return "gpt-4-mobile"
    if "gpt-4" in lowered:
        return "gpt-4"
    if "gpt-3.5" in lowered:
        return "text-davinci-002-render-sha"
    if "auto" in lowered:
        return "auto"
    return "gpt-4o"
