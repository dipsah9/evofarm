import React, { useState, useRef, useEffect } from 'react';
import { useAuth } from './AuthContext';
import './ChatWidget.css';

const RAG_URL = process.env.REACT_APP_RAG_URL || 'https://evofarm-rag.fly.dev';

function ChatWidget() {
  const { user, token } = useAuth();
  const [open, setOpen] = useState(false);
  const [messages, setMessages] = useState([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const messagesEndRef = useRef(null);
  const inputRef = useRef(null);

  // Auto-scroll to newest message
  useEffect(() => {
    if (messagesEndRef.current) {
      messagesEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages, loading]);

  // Auto-focus input when panel opens
  useEffect(() => {
    if (open && inputRef.current) {
      inputRef.current.focus();
    }
  }, [open]);

  // Only render for authenticated users
  if (!user) return null;

  const handleSend = async () => {
    const query = input.trim();
    if (!query || loading) return;

    // Add user message
    const userMessage = { role: 'user', text: query };
    setMessages((prev) => [...prev, userMessage]);
    setInput('');
    setLoading(true);

    try {
      const response = await fetch(`${RAG_URL}/api/v1/ask`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ query }),
      });

      if (!response.ok) {
        if (response.status === 401) {
          throw new Error('Your session expired. Please sign in again.');
        }
        if (response.status === 429) {
          throw new Error('Too many questions. Please wait a moment.');
        }
        throw new Error(`Request failed (${response.status})`);
      }

      const data = await response.json();

      setMessages((prev) => [
        ...prev,
        {
          role: 'bot',
          text: data.answer,
          sources: data.sources || [],
          confidence: data.confidence,
        },
      ]);
    } catch (err) {
      setMessages((prev) => [
        ...prev,
        {
          role: 'bot',
          text: `Sorry, something went wrong: ${err.message}`,
          error: true,
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const getInitialMessage = () => ({
    role: 'bot',
    text: `Hi ${user.name || 'there'}! I'm the EvoFarm documentation assistant. Ask me anything about the platform — architecture, solvers, deployment, how to add a new problem.`,
  });

  return (
    <>
      {/* Floating button */}
      {!open && (
        <button
          className="chat-widget-button"
          onClick={() => setOpen(true)}
          aria-label="Open documentation assistant"
        >
          💬
        </button>
      )}

      {/* Chat panel */}
      {open && (
        <div className="chat-widget-panel">
          <div className="chat-widget-header">
            <div className="chat-widget-title">
              <span className="chat-widget-icon">🧬</span>
              <span>Ask EvoFarm</span>
            </div>
            <button
              className="chat-widget-close"
              onClick={() => setOpen(false)}
              aria-label="Close chat"
            >
              ✕
            </button>
          </div>

          <div className="chat-widget-body">
            {/* Welcome message */}
            {messages.length === 0 && (
              <MessageBubble message={getInitialMessage()} />
            )}

            {/* Conversation */}
            {messages.map((msg, i) => (
              <MessageBubble key={i} message={msg} />
            ))}

            {/* Loading indicator */}
            {loading && (
              <div className="chat-widget-message chat-widget-message-bot">
                <div className="chat-widget-typing">
                  <span></span>
                  <span></span>
                  <span></span>
                </div>
              </div>
            )}

            <div ref={messagesEndRef} />
          </div>

          <div className="chat-widget-footer">
            <input
              ref={inputRef}
              type="text"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Ask about EvoFarm..."
              className="chat-widget-input"
              disabled={loading}
            />
            <button
              className="chat-widget-send"
              onClick={handleSend}
              disabled={loading || !input.trim()}
              aria-label="Send"
            >
              →
            </button>
          </div>
        </div>
      )}
    </>
  );
}

function MessageBubble({ message }) {
  const [sourcesOpen, setSourcesOpen] = useState(false);

  const isUser = message.role === 'user';
  const hasSources = message.sources && message.sources.length > 0;

  return (
    <div
      className={`chat-widget-message ${
        isUser ? 'chat-widget-message-user' : 'chat-widget-message-bot'
      } ${message.error ? 'chat-widget-message-error' : ''}`}
    >
      <div className="chat-widget-message-text">{message.text}</div>

      {hasSources && (
        <div className="chat-widget-sources">
          <button
            className="chat-widget-sources-toggle"
            onClick={() => setSourcesOpen(!sourcesOpen)}
          >
            {sourcesOpen ? '▾' : '▸'} {message.sources.length} source
            {message.sources.length === 1 ? '' : 's'}
          </button>

          {sourcesOpen && (
            <div className="chat-widget-sources-list">
              {message.sources.map((src, i) => {
                const filename =
                  src.metadata?.filename ||
                  src.metadata?.source_file ||
                  'document';
                return (
                  <div key={i} className="chat-widget-source">
                    <div className="chat-widget-source-file">
                      📄 {filename}
                    </div>
                    <div className="chat-widget-source-text">
                      {src.text?.substring(0, 160)}
                      {src.text?.length > 160 ? '…' : ''}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default ChatWidget;