"""
Ollama-based Chat Service for AquaFlow Analytics
Handles natural language query processing with local LLM and learning capabilities.
"""

import os
import json
import logging
import asyncio
from datetime import datetime
from typing import Dict, List, Optional, Any
from dataclasses import dataclass

import ollama
from fastapi import FastAPI, HTTPException, BackgroundTasks
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
import httpx

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Configuration
OLLAMA_HOST = os.getenv("OLLAMA_HOST", "http://ollama:11434")
BACKEND_URL = os.getenv("BACKEND_URL", "http://backend:3000")
MODEL_NAME = os.getenv("OLLAMA_MODEL", "tinyllama")

app = FastAPI(title="AquaFlow Chat Service", version="1.0.0")

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Pydantic models
class ChatMessage(BaseModel):
    session_id: str
    message: str
    user_id: Optional[str] = "default"

class ChatResponse(BaseModel):
    response: str
    session_id: str
    needs_clarification: bool = False
    clarification_question: Optional[str] = None
    entity_mappings: Dict[str, str] = {}
    confidence: float = 1.0

class LearningFeedback(BaseModel):
    session_id: str
    message_id: str
    user_query: str
    system_response: str
    user_correction: str
    is_helpful: bool

@dataclass
class ChatSession:
    session_id: str
    messages: List[Dict[str, Any]]
    entity_mappings: Dict[str, str]
    context: Dict[str, Any]
    last_activity: datetime

class ChatService:
    def __init__(self):
        self.sessions: Dict[str, ChatSession] = {}
        self.learning_cache = {}
        
    async def initialize_ollama(self):
        """Initialize Ollama client and ensure model is available"""
        try:
            # Check if Ollama is running
            client = ollama.Client(host=OLLAMA_HOST)
            models = client.list()
            
            # Check if our model is available
            available_models = [model['name'] for model in models['models']]
            if MODEL_NAME not in available_models:
                logger.info(f"Model {MODEL_NAME} not available yet. Will pull when needed.")
                # Don't block startup on model download
                
            logger.info(f"Ollama connection established")
            return True
            
        except Exception as e:
            logger.error(f"Failed to initialize Ollama: {e}")
            return False
    
    async def get_database_schema(self) -> Dict[str, Any]:
        """Fetch database schema from backend API"""
        try:
            async with httpx.AsyncClient() as client:
                response = await client.get(f"{BACKEND_URL}/api/schema")
                if response.status_code == 200:
                    return response.json()
                else:
                    logger.warning("Could not fetch schema from backend")
                    return {}
        except Exception as e:
            logger.error(f"Error fetching schema: {e}")
            return {}
    
    def create_system_prompt(self, schema_info: Dict[str, Any]) -> str:
        """Create system prompt with database schema context"""
        base_prompt = """You are Olivia, an AI assistant for AquaFlow Analytics - a water district operations platform.

Your role is to help water operations managers by:
1. Understanding natural language queries about water infrastructure
2. Converting them into appropriate database queries or actions
3. Asking clarifying questions when user intent is unclear

Database Context:
- Schema: aquaflow
- Core tables: datasets, parameters, series, numeric_values (TimescaleDB hypertable)
- Water infrastructure: canals, reservoirs, pump_stations
- Time-series data in numeric_values table

Common query patterns:
- "MC flow rate" likely refers to Main Canal flow rate parameter
- "reservoir levels" refers to water level measurements
- "pump station status" refers to operational status data

When you encounter ambiguous queries:
1. Ask specific clarifying questions
2. Suggest likely interpretations
3. Help users refine their requests

Always respond in a helpful, professional tone focused on water operations."""

        if schema_info:
            base_prompt += f"\n\nCurrent schema status: {schema_info.get('status', 'unknown')}"
            base_prompt += f"\nTotal tables: {schema_info.get('total_tables', 'unknown')}"
        
        return base_prompt
    
    async def process_query(self, message: str, session_id: str, user_id: str) -> ChatResponse:
        """Process user query with Ollama LLM"""
        try:
            # Get or create session
            if session_id not in self.sessions:
                schema_info = await self.get_database_schema()
                self.sessions[session_id] = ChatSession(
                    session_id=session_id,
                    messages=[],
                    entity_mappings={},
                    context={"schema": schema_info},
                    last_activity=datetime.now()
                )
            
            session = self.sessions[session_id]
            session.last_activity = datetime.now()
            
            # Build conversation context
            conversation = []
            
            # Add system prompt
            schema_info = session.context.get("schema", {})
            system_prompt = self.create_system_prompt(schema_info)
            conversation.append({"role": "system", "content": system_prompt})
            
            # Add conversation history (last 10 messages)
            for msg in session.messages[-10:]:
                conversation.append(msg)
            
            # Add current message
            conversation.append({"role": "user", "content": message})
            
            # Call Ollama
            try:
                client = ollama.Client(host=OLLAMA_HOST)
                
                # Check if model is available
                models = client.list()
                available_models = [model['name'] for model in models['models']]
                
                if MODEL_NAME not in available_models:
                    # Model not ready - return a helpful message
                    assistant_response = (
                        "I'm still downloading my language model. This may take a few minutes. "
                        "In the meantime, I can tell you that I'll be able to help you with:\n\n"
                        "• Checking water flow rates (e.g., 'MC flow rate')\n"
                        "• Monitoring pump stations\n"
                        "• Reviewing reservoir levels\n"
                        "• System status checks\n\n"
                        "Please try again in a moment!"
                    )
                else:
                    response = client.chat(
                        model=MODEL_NAME,
                        messages=conversation,
                        options={
                            "temperature": 0.7,
                            "top_p": 0.9,
                            "max_tokens": 500
                        }
                    )
                    assistant_response = response['message']['content']
                    
            except Exception as e:
                logger.error(f"Ollama chat error: {e}")
                assistant_response = (
                    "I'm having trouble connecting to my language model. "
                    "Please try again in a moment or contact support if the issue persists."
                )
            
            # Store messages in session
            session.messages.append({"role": "user", "content": message})
            session.messages.append({"role": "assistant", "content": assistant_response})
            
            # Analyze response for clarification needs
            needs_clarification = self._needs_clarification(assistant_response)
            clarification_question = self._extract_clarification(assistant_response) if needs_clarification else None
            
            return ChatResponse(
                response=assistant_response,
                session_id=session_id,
                needs_clarification=needs_clarification,
                clarification_question=clarification_question,
                entity_mappings=session.entity_mappings,
                confidence=0.8  # Default confidence
            )
            
        except Exception as e:
            logger.error(f"Error processing query: {e}")
            raise HTTPException(status_code=500, detail=f"Query processing failed: {str(e)}")
    
    def _needs_clarification(self, response: str) -> bool:
        """Detect if response indicates need for clarification"""
        clarification_indicators = [
            "could you clarify",
            "what do you mean by",
            "are you referring to",
            "which specific",
            "can you be more specific",
            "do you mean",
            "unclear"
        ]
        response_lower = response.lower()
        return any(indicator in response_lower for indicator in clarification_indicators)
    
    def _extract_clarification(self, response: str) -> Optional[str]:
        """Extract clarification question from response"""
        lines = response.split('\n')
        for line in lines:
            if '?' in line and any(word in line.lower() for word in ['clarify', 'mean', 'specific', 'referring']):
                return line.strip()
        return None
    
    async def learn_from_feedback(self, feedback: LearningFeedback):
        """Learn from user feedback and corrections"""
        try:
            # Store learning feedback
            learning_key = f"{feedback.session_id}_{feedback.message_id}"
            self.learning_cache[learning_key] = {
                "user_query": feedback.user_query,
                "system_response": feedback.system_response,
                "user_correction": feedback.user_correction,
                "is_helpful": feedback.is_helpful,
                "timestamp": datetime.now().isoformat()
            }
            
            # Update entity mappings if applicable
            if feedback.session_id in self.sessions:
                session = self.sessions[feedback.session_id]
                # Simple entity extraction from correction
                if "means" in feedback.user_correction.lower():
                    parts = feedback.user_correction.lower().split("means")
                    if len(parts) == 2:
                        entity = parts[0].strip().strip('"\'')
                        mapping = parts[1].strip().strip('"\'')
                        session.entity_mappings[entity] = mapping
            
            logger.info(f"Learned from feedback for session {feedback.session_id}")
            
        except Exception as e:
            logger.error(f"Error processing feedback: {e}")

# Initialize chat service
chat_service = ChatService()

# API Endpoints
@app.on_event("startup")
async def startup_event():
    """Initialize services on startup"""
    success = await chat_service.initialize_ollama()
    if not success:
        logger.warning("Ollama initialization failed - service may have limited functionality")

@app.get("/health")
async def health_check():
    """Health check endpoint"""
    try:
        # Check Ollama connectivity
        client = ollama.Client(host=OLLAMA_HOST)
        models = client.list()
        
        return {
            "status": "healthy",
            "timestamp": datetime.now().isoformat(),
            "service": "aquaflow-chat-service",
            "ollama_status": "connected",
            "model": MODEL_NAME,
            "available_models": len(models.get('models', []))
        }
    except Exception as e:
        return {
            "status": "degraded",
            "timestamp": datetime.now().isoformat(),
            "service": "aquaflow-chat-service",
            "ollama_status": "error",
            "error": str(e)
        }

@app.post("/chat", response_model=ChatResponse)
async def chat_endpoint(message: ChatMessage, background_tasks: BackgroundTasks):
    """Main chat endpoint"""
    response = await chat_service.process_query(
        message.message,
        message.session_id,
        message.user_id or "default"
    )
    return response

@app.post("/feedback")
async def feedback_endpoint(feedback: LearningFeedback, background_tasks: BackgroundTasks):
    """Learning feedback endpoint"""
    background_tasks.add_task(chat_service.learn_from_feedback, feedback)
    return {"status": "feedback_received", "session_id": feedback.session_id}

@app.get("/sessions/{session_id}")
async def get_session(session_id: str):
    """Get session information"""
    if session_id not in chat_service.sessions:
        raise HTTPException(status_code=404, detail="Session not found")
    
    session = chat_service.sessions[session_id]
    return {
        "session_id": session.session_id,
        "message_count": len(session.messages),
        "entity_mappings": session.entity_mappings,
        "last_activity": session.last_activity.isoformat()
    }

@app.delete("/sessions/{session_id}")
async def clear_session(session_id: str):
    """Clear session data"""
    if session_id in chat_service.sessions:
        del chat_service.sessions[session_id]
        return {"status": "session_cleared", "session_id": session_id}
    else:
        raise HTTPException(status_code=404, detail="Session not found")

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=8001,
        reload=True,
        log_level="info"
    )