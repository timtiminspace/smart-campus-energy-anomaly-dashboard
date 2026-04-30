# syntax=docker/dockerfile:1.7

FROM python:3.11-slim AS runtime

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

WORKDIR /app

# CPU-only PyTorch wheels — keeps the image ~2 GB instead of pulling CUDA.
COPY requirements.txt ./
RUN pip install --extra-index-url https://download.pytorch.org/whl/cpu \
        -r requirements.txt

# Brain code + trained weights + scaler.
COPY brain.py scaler.json best_model.pth ./

RUN useradd --create-home --uid 1000 app && chown -R app:app /app
USER app

EXPOSE 8000

HEALTHCHECK --interval=20s --timeout=5s --start-period=30s --retries=3 \
  CMD python -c "import urllib.request,sys; urllib.request.urlopen('http://127.0.0.1:8000/docs',timeout=3)" || exit 1

CMD ["uvicorn", "brain:app", "--host", "0.0.0.0", "--port", "8000"]
