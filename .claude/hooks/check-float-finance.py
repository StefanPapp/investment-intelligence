import json
import sys
import re


data = json.load(sys.stdin)
tool_input = data.get("tool_input", {})
file_path = tool_input.get("file_path", "")
content = tool_input.get("content", "") or tool_input.get("new_str", "")


# Only check files in financial code paths
finance_paths = ["portfolio", "returns", "transaction", "valuation", "price"]
if not any(p in file_path.lower() for p in finance_paths):
    sys.exit(0)  # Not a financial file, allow the edit


# Check for floating-point usage in monetary contexts
float_patterns = [
    r"\\bfloat\\b",
    r"\\bfloat32\\b",
    r"\\bfloat64\\b",
    r"\\b\\d+\\.\\d+\\s*[\\+\\-\\*\\/]",  # literal float arithmetic
]


for pattern in float_patterns:
    if re.search(pattern, content):
        print(
            "BLOCKED: Floating-point arithmetic detected in financial code. "
            "Use Decimal types for all monetary calculations. "
            f"Pattern found in: {file_path}",
            file=sys.stderr,
        )
        sys.exit(2)  # Exit code 2 blocks the operation


sys.exit(0)  # All checks passed, allow the edit
