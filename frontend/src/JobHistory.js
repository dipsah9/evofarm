import React, { useState, useEffect } from 'react';
import api from './api';
function JobHistory({ onSelectJob }) {
  const [jobs, setJobs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    const fetchHistory = async () => {
      try {
        const res = await api.get('/jobs/history?limit=50');
        setJobs(res.data);
        setError('');
      } catch (err) {
        console.error('History fetch failed:', err);
        setError('Failed to load history');
      } finally {
        setLoading(false);
      }
    };

    fetchHistory();
    const interval = setInterval(fetchHistory, 5000);
    return () => clearInterval(interval);
  }, []);

  if (loading) return <div className="history-empty">Loading...</div>;
  if (error) return <div className="history-empty">{error}</div>;
  if (jobs.length === 0) return <div className="history-empty">No jobs yet</div>;

  const getStatusIcon = (status) => {
    switch (status) {
      case 'completed': return '✅';
      case 'running': return '🔄';
      case 'failed': return '❌';
      case 'pending': return '⏳';
      default: return '·';
    }
  };

  return (
    <div className="history-list">
      {jobs.map((job) => (
        <div
          key={job.id}
          className="history-row"
          onClick={() => onSelectJob(job.id)}
        >
          <div className="history-icon">{getStatusIcon(job.status)}</div>
          <div className="history-main">
            <div className="history-id">{job.id.substring(0, 8)}…</div>
            <div className="history-meta">
              {job.problem} · {job.solver_type || 'auto'}
            </div>
          </div>
          <div className="history-fitness">
            {job.best_fitness != null ? job.best_fitness.toFixed(4) : '—'}
          </div>
          <div className="history-status">{job.status}</div>
        </div>
      ))}
    </div>
  );
}

export default JobHistory;