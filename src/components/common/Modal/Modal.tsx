import React, { useEffect, useRef } from 'react';
import { createPortal } from 'react-dom';
import styles from '../../../styles/components/Modal.module.css';

export interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl' | 'full';
  variant?: 'default' | 'glassmorphism' | 'primary' | 'success' | 'warning' | 'error';
  showCloseButton?: boolean;
  closeOnEscapeKey?: boolean;
  closeOnOverlayClick?: boolean;
  scrollable?: boolean;
  loading?: boolean;
  className?: string;
  'data-testid'?: string;
}

const Modal: React.FC<ModalProps> = ({
  isOpen,
  onClose,
  title,
  children,
  footer,
  size = 'md',
  variant = 'default',
  showCloseButton = true,
  closeOnEscapeKey = true,
  closeOnOverlayClick = true,
  scrollable = false,
  loading = false,
  className,
  'data-testid': testId,
}) => {
  const modalRef = useRef<HTMLDivElement>(null);
  const previousActiveElement = useRef<HTMLElement | null>(null);

  // Handle escape key
  useEffect(() => {
    if (!closeOnEscapeKey || !isOpen) return;

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [isOpen, closeOnEscapeKey, onClose]);

  // Handle focus management
  useEffect(() => {
    if (isOpen) {
      previousActiveElement.current = document.activeElement as HTMLElement;
      
      // Focus the modal after a brief delay to ensure it's rendered
      setTimeout(() => {
        if (modalRef.current) {
          modalRef.current.focus();
        }
      }, 100);
    } else if (previousActiveElement.current) {
      previousActiveElement.current.focus();
    }
  }, [isOpen]);

  // Prevent body scroll when modal is open
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }

    return () => {
      document.body.style.overflow = '';
    };
  }, [isOpen]);

  const handleOverlayClick = (event: React.MouseEvent) => {
    if (closeOnOverlayClick && event.target === event.currentTarget) {
      onClose();
    }
  };

  const modalClassNames = [
    styles.modal,
    styles[size],
    variant !== 'default' && styles[variant],
    loading && styles.loading,
    className
  ].filter(Boolean).join(' ');

  const overlayClassNames = [
    styles.overlay,
    isOpen && styles.open
  ].filter(Boolean).join(' ');

  if (!isOpen) return null;

  const modalContent = (
    <div 
      className={overlayClassNames}
      onClick={handleOverlayClick}
      role="dialog"
      aria-modal="true"
      aria-labelledby={title ? 'modal-title' : undefined}
      data-testid={testId}
    >
      <div 
        ref={modalRef}
        className={modalClassNames}
        tabIndex={-1}
      >
        {(title || showCloseButton) && (
          <div className={styles.header}>
            {title && (
              <h2 id="modal-title" className={styles.title}>
                {title}
              </h2>
            )}
            {showCloseButton && (
              <button 
                className={styles.closeButton}
                onClick={onClose}
                aria-label="Close modal"
                type="button"
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <line x1="18" y1="6" x2="6" y2="18" />
                  <line x1="6" y1="6" x2="18" y2="18" />
                </svg>
              </button>
            )}
          </div>
        )}
        
        <div className={scrollable ? styles.scrollableBody : styles.body}>
          {children}
        </div>
        
        {footer && (
          <div className={styles.footer}>
            {footer}
          </div>
        )}
      </div>
    </div>
  );

  // Render modal in a portal
  return createPortal(modalContent, document.body);
};

export default Modal;