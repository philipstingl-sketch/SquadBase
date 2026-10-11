"""Compatibility entry point; the current app uses the v4.2 navigation layer."""
from pathlib import Path
import runpy

runpy.run_path(str(Path(__file__).with_name("browser_v43_test.py")), run_name="__main__")
