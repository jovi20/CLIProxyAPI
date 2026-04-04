import unittest

from chatgpt.model_settings import extract_reasoning_effort, resolve_request_model, split_model_name_and_suffix


class ModelSettingsTests(unittest.TestCase):
    def test_split_model_name_and_suffix(self):
        base, suffix = split_model_name_and_suffix("gpt-5.4(xhigh)")
        self.assertEqual(base, "gpt-5.4")
        self.assertEqual(suffix, "xhigh")

    def test_split_model_name_without_suffix(self):
        base, suffix = split_model_name_and_suffix("gpt-5.4")
        self.assertEqual(base, "gpt-5.4")
        self.assertEqual(suffix, "")

    def test_resolve_request_model_keeps_latest_gpt5(self):
        self.assertEqual(resolve_request_model("gpt-5.4"), "gpt-5.4")

    def test_resolve_request_model_maps_codex_bridge_aliases(self):
        self.assertEqual(resolve_request_model("gpt-5.3-codex"), "gpt-5.4")
        self.assertEqual(resolve_request_model("gpt-5.1-codex"), "gpt-5.3")

    def test_resolve_request_model_preserves_existing_openai_models(self):
        self.assertEqual(resolve_request_model("gpt-4o-mini"), "gpt-4o-mini")
        self.assertEqual(resolve_request_model("o3-mini-high"), "o3-mini-high")

    def test_extract_reasoning_effort_from_openai_payload(self):
        payload = {"reasoning_effort": "xhigh"}
        self.assertEqual(extract_reasoning_effort(payload, ""), "xhigh")

    def test_extract_reasoning_effort_from_nested_reasoning_payload(self):
        payload = {"reasoning": {"effort": "high"}}
        self.assertEqual(extract_reasoning_effort(payload, ""), "high")

    def test_extract_reasoning_effort_falls_back_to_model_suffix(self):
        self.assertEqual(extract_reasoning_effort({}, "medium"), "medium")


if __name__ == "__main__":
    unittest.main()
