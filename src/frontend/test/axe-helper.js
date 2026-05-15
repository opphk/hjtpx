import React from 'react';
import { render } from '@testing-library/react';

const axe = async (container) => {
  const { default: axeCore } = await import('axe-core');
  
  return new Promise((resolve, reject) => {
    axeCore.run(container, {
      runOnly: {
        type: 'tag',
        values: ['wcag2a', 'wcag2aa', 'wcag21aa']
      }
    }, (err, results) => {
      if (err) {
        reject(err);
      } else {
        resolve(results);
      }
    });
  });
};

export const axeTest = async (component, options = {}) => {
  const { container } = render(component);
  
  try {
    const results = await axe(container);
    
    if (results.violations.length > 0) {
      const violations = results.violations.map(v => ({
        id: v.id,
        impact: v.impact,
        description: v.description,
        help: v.helpUrl,
        nodes: v.nodes.map(n => ({
          html: n.html,
          target: n.target,
          failureSummary: n.failureSummary
        }))
      }));
      
      throw new Error(
        `Accessibility violations found:\n${JSON.stringify(violations, null, 2)}`
      );
    }
    
    return results;
  } catch (error) {
    throw error;
  }
};

export const checkAccessibility = async (container) => {
  const results = await axe(container);
  
  if (results.violations.length > 0) {
    const errorMessages = results.violations.map(v => {
      return `${v.id} (${v.impact}): ${v.description}\n${v.nodes.map(n => `  - ${n.target}: ${n.failureSummary}`).join('\n')}`;
    }).join('\n\n');
    
    throw new Error(`Accessibility violations:\n${errorMessages}`);
  }
  
  return results;
};

export default axe;
