import React from 'react';

interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  label: string;
}

export const Select: React.FC<SelectProps> = ({
  label,
  children,
  ...props
}) => {
  return (
    <div className="form-group">
      <label>{label}</label>
      <select className="form-select" {...props}>
        {children}
      </select>
    </div>
  );
};
