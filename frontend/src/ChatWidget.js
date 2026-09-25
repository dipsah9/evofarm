import React, { useState } from 'react';
import './ChatWidget.css';

function ChatWidget() {
  const [open, setOpen] = useState(false);

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
          <span className="chat-widget-badge">Coming soon</span>
        </button>
      )}

      {/* Chat panel */}
      {open && (
        <div className="chat-widget-panel">
          <div className="chat-widget-header">
            <div className="chat-widget-title">
              <span className="chat-widget-icon">🧬</span>
              <span>EvoFarm Assistant</span>
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
            <div className="chat-widget-message-bot">
              <p>
                Hi! I'm the EvoFarm documentation assistant.
              </p>
              <p>
                Soon, you'll be able to ask me anything about EvoFarm —
                the architecture, the solvers, how to deploy it, how to
                add a new problem.
              </p>
              <p className="chat-widget-coming-soon">
                <strong>🚧 Coming soon</strong>
              </p>
              <p className="chat-widget-note">
                This assistant will use RAG to answer questions grounded
                in EvoFarm's documentation. No hallucination — just
                cited answers from the docs.
              </p>
            </div>
          </div>

          <div className="chat-widget-footer">
            <input
              type="text"
              placeholder="Ask a question (coming soon)"
              disabled
              className="chat-widget-input"
            />
            <button
              className="chat-widget-send"
              disabled
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

export default ChatWidget;