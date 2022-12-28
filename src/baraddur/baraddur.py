import logging
from sys import stderr, stdout


def _setup_logger(name: str, log_level: int) -> logging.Logger:
    class _InfoFilter(logging.Filter):
        def filter(self: "_InfoFilter", rec: logging.LogRecord) -> bool:
            return rec.levelno in (logging.DEBUG, logging.INFO)

    logger = logging.getLogger(name)

    logger.setLevel(log_level)

    stdout_handler = logging.StreamHandler(stdout)
    stdout_handler.setLevel(log_level)
    stdout_handler.addFilter(_InfoFilter())
    stderr_handler = logging.StreamHandler(stderr)
    stderr_handler.setLevel(logging.WARNING)

    logger.addHandler(stdout_handler)
    logger.addHandler(stderr_handler)

    return logger


def scan():
    raise NotImplementedError()


def main():
    raise NotImplementedError()
