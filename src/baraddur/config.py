from pathlib import Path
from typing import List, Optional, Pattern

import yaml
from pydantic import BaseModel, ValidationError


class ScanSettings(BaseModel):
    regex: Pattern
    interupt_when_matched: Optional[bool] = False


class WorkSettings(BaseModel):
    script: str


class JobSettings(BaseModel):
    scan: ScanSettings
    work: WorkSettings


class Settings(BaseModel):
    scanner_concurrency: int
    worker_concurrency: int
    jobsSettings: List[JobSettings]


def load_config(config_path: Path) -> Settings:
    try:
        raw_config = yaml.safe_load(config_path)
        return Settings.parse_obj(raw_config)
    except FileNotFoundError:
        pass
    except yaml.scanner.ScannerError:
        pass
    except yaml.error.YAMLError:
        pass
    except ValidationError:
        pass
