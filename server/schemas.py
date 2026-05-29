from datetime import datetime

from pydantic import BaseModel, Field


class SpeciesInfo(BaseModel):
    id: str
    name: str
    color: str


class SimConfigCreate(BaseModel):
    title: str = Field(max_length=80)
    description: str = Field(default="", max_length=400)
    author: str = Field(default="anonymous", max_length=60)


class SimConfigResponse(BaseModel):
    id: str
    title: str
    description: str
    author: str
    original_filename: str
    file_type: str
    file_size: int
    species: list[SpeciesInfo]
    robot_count: int
    arena_width: float
    arena_height: float
    download_count: int
    created_at: datetime

    model_config = {"from_attributes": True}


class SimConfigListItem(BaseModel):
    id: str
    title: str
    description: str
    author: str
    file_type: str
    species: list[SpeciesInfo]
    robot_count: int
    download_count: int
    created_at: datetime

    model_config = {"from_attributes": True}


class PaginatedConfigs(BaseModel):
    items: list[SimConfigListItem]
    total: int
    page: int
    page_size: int


class LeaderboardSubmit(BaseModel):
    player_id: str = Field(min_length=36, max_length=36)
    username: str = Field(min_length=1, max_length=30)
    time_seconds: float = Field(ge=2.0, le=90.0)
    algorithm: list = Field(default_factory=list)

class LeaderboardResponse(BaseModel):
    id: str
    username: str
    time_seconds: float
    created_at: datetime

    model_config = {"from_attributes": True}
