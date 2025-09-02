import React from 'react';
import styles from '../../../styles/components/Card.module.css';

export interface CardProps {
  children: React.ReactNode;
  variant?: 'default' | 'elevated' | 'flat' | 'glass' | 'gradient';
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  color?: 'default' | 'primary' | 'success' | 'warning' | 'error';
  clickable?: boolean;
  selected?: boolean;
  loading?: boolean;
  animated?: boolean;
  className?: string;
  onClick?: () => void;
  'data-testid'?: string;
}

export interface CardHeaderProps {
  children: React.ReactNode;
  className?: string;
}

export interface CardBodyProps {
  children: React.ReactNode;
  className?: string;
}

export interface CardFooterProps {
  children: React.ReactNode;
  className?: string;
}

export interface CardTitleProps {
  children: React.ReactNode;
  className?: string;
}

export interface CardSubtitleProps {
  children: React.ReactNode;
  className?: string;
}

export interface CardDescriptionProps {
  children: React.ReactNode;
  className?: string;
}

const Card: React.FC<CardProps> = ({
  children,
  variant = 'default',
  size = 'md',
  color = 'default',
  clickable = false,
  selected = false,
  loading = false,
  animated = false,
  className,
  onClick,
  'data-testid': testId,
  ...props
}) => {
  const classNames = [
    styles.card,
    variant !== 'default' && styles[variant],
    size !== 'md' && styles[size],
    color !== 'default' && styles[color],
    clickable && styles.clickable,
    selected && styles.selected,
    loading && styles.loading,
    animated && styles.animated,
    className
  ].filter(Boolean).join(' ');

  return (
    <div
      className={classNames}
      onClick={clickable ? onClick : undefined}
      data-testid={testId}
      {...props}
    >
      {children}
    </div>
  );
};

export const CardHeader: React.FC<CardHeaderProps> = ({ children, className }) => (
  <div className={[styles.cardHeader, className].filter(Boolean).join(' ')}>
    {children}
  </div>
);

export const CardBody: React.FC<CardBodyProps> = ({ children, className }) => (
  <div className={[styles.cardBody, className].filter(Boolean).join(' ')}>
    {children}
  </div>
);

export const CardFooter: React.FC<CardFooterProps> = ({ children, className }) => (
  <div className={[styles.cardFooter, className].filter(Boolean).join(' ')}>
    {children}
  </div>
);

export const CardTitle: React.FC<CardTitleProps> = ({ children, className }) => (
  <h3 className={[styles.cardTitle, className].filter(Boolean).join(' ')}>
    {children}
  </h3>
);

export const CardSubtitle: React.FC<CardSubtitleProps> = ({ children, className }) => (
  <p className={[styles.cardSubtitle, className].filter(Boolean).join(' ')}>
    {children}
  </p>
);

export const CardDescription: React.FC<CardDescriptionProps> = ({ children, className }) => (
  <p className={[styles.cardDescription, className].filter(Boolean).join(' ')}>
    {children}
  </p>
);

export default Card;
