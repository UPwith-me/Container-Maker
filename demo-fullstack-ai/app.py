"""
🚀 Container-Maker Demo: FastAPI Application
This demo shows cm can run a real Python web server.
"""

from fastapi import FastAPI
from fastapi.responses import HTMLResponse
import platform
import sys
import os

app = FastAPI(title="Container-Maker Demo")

@app.get("/", response_class=HTMLResponse)
async def root():
    return """
    <!DOCTYPE html>
    <html>
    <head>
        <title>Container-Maker Demo</title>
        <style>
            body { 
                font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                min-height: 100vh;
                display: flex;
                align-items: center;
                justify-content: center;
                margin: 0;
            }
            .card {
                background: white;
                border-radius: 20px;
                padding: 40px;
                box-shadow: 0 20px 60px rgba(0,0,0,0.3);
                max-width: 600px;
                text-align: center;
            }
            h1 { color: #667eea; margin-bottom: 10px; }
            .emoji { font-size: 48px; }
            .info { background: #f5f5f5; padding: 15px; border-radius: 10px; margin: 20px 0; text-align: left; }
            .success { color: #22c55e; font-weight: bold; }
        </style>
    </head>
    <body>
        <div class="card">
            <div class="emoji">🚀</div>
            <h1>Container-Maker Demo</h1>
            <p class="success">✅ Running inside container!</p>
            <div class="info">
                <strong>Environment Info:</strong><br>
                Python: """ + sys.version.split()[0] + """<br>
                Platform: """ + platform.platform() + """<br>
                Container: """ + os.environ.get('HOSTNAME', 'unknown')[:12] + """
            </div>
            <p>This FastAPI server was started with <code>cm run</code></p>
        </div>
    </body>
    </html>
    """

@app.get("/api/info")
async def info():
    return {
        "status": "running",
        "python_version": sys.version,
        "platform": platform.platform(),
        "container_id": os.environ.get('HOSTNAME', 'unknown')[:12],
        "message": "Hello from Container-Maker! 🎉"
    }

if __name__ == "__main__":
    import uvicorn
    print("🚀 Starting FastAPI server on http://localhost:8000")
    uvicorn.run(app, host="0.0.0.0", port=8000)
