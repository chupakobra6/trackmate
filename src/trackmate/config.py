from functools import lru_cache

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", env_prefix="TRACKMATE__", extra="ignore")

    bot_token: str
    database_url: str
    default_timezone: str = Field(default="Europe/Moscow")
    worker_tick_seconds: int = Field(default=5)
    material_batch_timeout_seconds: int = Field(default=15)
    log_level: str = Field(default="INFO")


@lru_cache(maxsize=1)
def get_settings() -> Settings:
    return Settings()
