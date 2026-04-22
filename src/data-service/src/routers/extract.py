import logging
import tempfile
from pathlib import Path

from fastapi import APIRouter, HTTPException, UploadFile

from src.models.extract import ExtractionResult
from src.services.extraction_service import ExtractionService

logger = logging.getLogger(__name__)

router = APIRouter()

try:
    extraction_service = ExtractionService()
except ValueError:
    extraction_service = None
    logger.warning("ANTHROPIC_API_KEY not configured — /extract will return 503")


@router.post("/extract", response_model=ExtractionResult)
async def extract(file: UploadFile) -> ExtractionResult:
    """Extract transactions from an uploaded broker statement file."""
    if extraction_service is None:
        raise HTTPException(status_code=503, detail="ANTHROPIC_API_KEY not configured")

    if file.filename is None:
        raise HTTPException(status_code=400, detail="filename required")

    ext = Path(file.filename).suffix.lower()
    content_bytes = await file.read()

    if ext == ".csv":
        text = content_bytes.decode("utf-8", errors="replace")
        return await extraction_service.extract_csv(text)

    if ext in (".png", ".jpg", ".jpeg", ".pdf"):
        with tempfile.NamedTemporaryFile(suffix=ext, delete=False) as tmp:
            tmp.write(content_bytes)
            tmp_path = tmp.name
        try:
            return await extraction_service.extract_image(tmp_path)
        finally:
            Path(tmp_path).unlink(missing_ok=True)

    raise HTTPException(status_code=400, detail=f"unsupported file type: {ext}")
