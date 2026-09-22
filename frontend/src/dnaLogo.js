import React from 'react';
import './DnaLogo.css';

/**
 * A rotating DNA double helix.
 * 
 * Props:
 *   size      — pixel size (width and height)
 *   speed     — animation duration in seconds
 *   colorA    — first strand color
 *   colorB    — second strand color
 *   rungs     — number of base pairs (default 10)
 */
function DnaLogo({
  size = 48,
  speed = 12,
  colorA = '#667eea',
  colorB = '#764ba2',
  rungs = 10,
}) {
  const strands = Array.from({ length: rungs });

  return (
    <div
      className="dna-logo"
      style={{
        '--size': `${size}px`,
        '--speed': `${speed}s`,
        '--color-a': colorA,
        '--color-b': colorB,
      }}
    >
      {strands.map((_, i) => (
        <div
          key={i}
          className="dna-rung"
          style={{
            '--i': i,
            '--total': rungs,
          }}
        >
          <span className="dna-node dna-node-a" />
          <span className="dna-bar" />
          <span className="dna-node dna-node-b" />
        </div>
      ))}
    </div>
  );
}

export default DnaLogo;