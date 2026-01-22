import React from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import type { Components } from 'react-markdown';

const markdownComponents: Components = {
  pre: ({ children }) => (
    <pre style={{
      background: '#f0f0f0',
      padding: 12,
      borderRadius: 4,
      overflow: 'auto',
      fontSize: 13,
    }}>
      {children}
    </pre>
  ),
  code: ({ children, className }) => {
    const isInline = !className;
    return isInline ? (
      <code style={{
        background: '#f0f0f0',
        padding: '2px 6px',
        borderRadius: 3,
        fontSize: 13,
      }}>
        {children}
      </code>
    ) : (
      <code style={{ fontSize: 13 }}>{children}</code>
    );
  },
  table: ({ children }) => (
    <table style={{
      borderCollapse: 'collapse',
      width: '100%',
      marginBottom: 16,
    }}>
      {children}
    </table>
  ),
  th: ({ children }) => (
    <th style={{
      border: '1px solid #d9d9d9',
      padding: '8px 12px',
      background: '#fafafa',
      textAlign: 'left',
    }}>
      {children}
    </th>
  ),
  td: ({ children }) => (
    <td style={{
      border: '1px solid #d9d9d9',
      padding: '8px 12px',
    }}>
      {children}
    </td>
  ),
  ul: ({ children }) => (
    <ul style={{ paddingLeft: 20, marginBottom: 8 }}>{children}</ul>
  ),
  ol: ({ children }) => (
    <ol style={{ paddingLeft: 20, marginBottom: 8 }}>{children}</ol>
  ),
  li: ({ children }) => (
    <li style={{ marginBottom: 4 }}>{children}</li>
  ),
  h1: ({ children }) => (
    <h1 style={{ fontSize: 20, fontWeight: 600, marginBottom: 12, marginTop: 16 }}>{children}</h1>
  ),
  h2: ({ children }) => (
    <h2 style={{ fontSize: 18, fontWeight: 600, marginBottom: 10, marginTop: 14 }}>{children}</h2>
  ),
  h3: ({ children }) => (
    <h3 style={{ fontSize: 16, fontWeight: 600, marginBottom: 8, marginTop: 12 }}>{children}</h3>
  ),
  p: ({ children }) => (
    <p style={{ marginBottom: 8 }}>{children}</p>
  ),
  blockquote: ({ children }) => (
    <blockquote style={{
      borderLeft: '4px solid #d9d9d9',
      paddingLeft: 16,
      margin: '8px 0',
      color: '#666',
    }}>
      {children}
    </blockquote>
  ),
};

interface MarkdownContentProps {
  content: string;
  className?: string;
  style?: React.CSSProperties;
}

const MarkdownContent: React.FC<MarkdownContentProps> = ({ content, className, style }) => {
  return (
    <div
      className={className}
      style={{
        padding: 16,
        background: '#fafafa',
        borderRadius: 4,
        lineHeight: 1.6,
        ...style,
      }}
    >
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={markdownComponents}>
        {content}
      </ReactMarkdown>
    </div>
  );
};

export default MarkdownContent;
