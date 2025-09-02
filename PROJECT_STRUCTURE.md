# FluentFox - Improved Project Structure

## Overview

This document outlines the restructured project organization focused on maintainability, scalability, and consistent design patterns.

## 🎨 Design System

### Core Principles
- **Consistency**: Unified design tokens and components
- **Accessibility**: WCAG 2.1 AA compliant
- **Performance**: Optimized CSS and minimal redundancy
- **Maintainability**: Centralized styles and clear organization

## 📁 New File Structure

```
src/
├── styles/
│   ├── index.css              # Main entry point for all styles
│   ├── variables.css          # Design system tokens
│   ├── reset.css              # Modern CSS reset
│   ├── globals.css            # Global styles and utilities
│   ├── components/            # Component-specific styles
│   │   ├── Button.module.css
│   │   ├── Modal.module.css
│   │   ├── Card.module.css
│   │   └── Input.module.css
│   ├── layout/                # Layout component styles
│   │   ├── Navbar.module.css
│   │   ├── Footer.module.css
│   │   ├── Header.module.css
│   │   └── Sidebar.module.css
│   ├── pages/                 # Page-specific styles
│   │   ├── Home.module.css
│   │   ├── AlphabetChart.module.css
│   │   └── [page].module.css
│   └── utilities/             # Utility classes
│       └── utilities.css      # Tailwind-like utilities
├── components/
│   ├── common/                # Reusable UI components
│   │   ├── Button/
│   │   │   ├── Button.tsx     # Enhanced component
│   │   │   ├── Button.module.css (deprecated)
│   │   │   └── index.ts
│   │   ├── Modal/
│   │   ├── Card/
│   │   └── Input/
│   ├── layout/                # Layout components
│   └── ui/                    # App-specific UI components
└── ...
```

## 🔧 Key Improvements

### 1. Centralized Styles
- All CSS is now organized in `src/styles/`
- Component styles moved from component folders to centralized locations
- Easier to find and maintain styles
- Reduced duplication and conflicts

### 2. Design System
- Comprehensive design tokens in `variables.css`
- Consistent spacing, colors, typography, and shadows
- Modern CSS custom properties
- Backward compatibility with existing code

### 3. Enhanced Components

#### Button Component
- Multiple variants: `primary`, `secondary`, `outline`, `ghost`, etc.
- Learning type variants: `kanji`, `hiragana`, `katakana`, `grammar`, `jlpt`
- Size options: `xs`, `sm`, `md`, `lg`, `xl`
- States: `loading`, `disabled`, `selected`
- Accessibility features and keyboard navigation

#### Modal Component
- Portal-based rendering
- Multiple sizes and variants
- Accessibility features (focus management, keyboard navigation)
- Animation and backdrop effects
- Mobile-responsive design

#### Card Component
- Flexible layout options
- Multiple variants and states
- Hover effects and animations
- Loading states
- Responsive design

### 4. Utility Classes
- Tailwind-like utility system
- Spacing, typography, layout utilities
- Responsive variants
- Performance optimized

## 🎯 Usage Examples

### Importing Styles
```tsx
// Main app entry point
import './styles/index.css';

// Component-specific styles
import styles from '../styles/components/Button.module.css';
import styles from '../styles/layout/Navbar.module.css';
import styles from '../styles/pages/Home.module.css';
```

### Using Components
```tsx
import Button from './components/common/Button';
import { Card, CardHeader, CardBody, CardFooter } from './components/common/Card';
import Modal from './components/common/Modal';

// Button usage
<Button variant="primary" size="lg" leftIcon={<Icon />}>
  Click me
</Button>

// Card usage
<Card variant="elevated" clickable>
  <CardHeader>
    <CardTitle>Title</CardTitle>
  </CardHeader>
  <CardBody>
    Content here
  </CardBody>
</Card>

// Modal usage
<Modal
  isOpen={isOpen}
  onClose={() => setIsOpen(false)}
  title="Modal Title"
  size="lg"
>
  Modal content
</Modal>
```

### Using Utilities
```tsx
<div className="flex items-center justify-between p-4 bg-white rounded-lg shadow-md">
  <h3 className="text-xl font-semibold text-gray-900">Title</h3>
  <Button variant="primary">Action</Button>
</div>
```

## 🎨 Design Tokens

### Colors
- Primary: `--color-primary-[50-900]`
- Secondary: `--color-secondary-[50-900]`
- Gray: `--color-gray-[50-900]`
- Semantic: `--color-success`, `--color-warning`, `--color-error`
- Learning types: `--color-kanji`, `--color-hiragana`, etc.

### Spacing
- Scale: `--space-[0-32]` (0px to 128px)
- Named: `--space-xs`, `--space-sm`, `--space-md`, etc.

### Typography
- Sizes: `--font-size-[xs-7xl]`
- Weights: `--font-weight-[thin-black]`
- Line heights: `--line-height-[tight-loose]`

### Shadows
- Base: `--shadow-[sm-2xl]`
- Colored: `--shadow-primary`, `--shadow-success`, etc.

## 🚀 Migration Guide

### From Old Structure
1. Update import paths in components
2. Replace old CSS modules with centralized versions
3. Use new component props instead of custom CSS
4. Leverage utility classes for common patterns

### Breaking Changes
- Component CSS module paths changed
- Some CSS class names updated
- Design tokens replaced old CSS variables

## 📱 Responsive Design

All components and utilities include responsive variants:
- Mobile-first approach
- Breakpoints: `sm` (640px), `md` (768px), `lg` (1024px), `xl` (1280px)
- Utility classes with responsive prefixes: `sm:`, `md:`, `lg:`

## ♿ Accessibility

Enhanced accessibility features:
- Proper ARIA attributes
- Keyboard navigation
- Focus management
- Screen reader support
- High contrast support
- Reduced motion preferences

## 🔄 Performance

Optimizations implemented:
- CSS-in-JS eliminated for better performance
- Reduced bundle size through tree shaking
- Optimized animations and transitions
- Lazy loading for non-critical styles

## 🧪 Testing

All components include:
- TypeScript definitions
- PropTypes (where applicable)
- Test IDs for automated testing
- Storybook stories (recommended)

## 📚 Best Practices

1. **Always use the design system tokens**
2. **Prefer utility classes for simple styling**
3. **Use component variants instead of custom CSS**
4. **Follow the established naming conventions**
5. **Maintain consistency across the application**

## 🔮 Future Enhancements

Planned improvements:
- Dark mode support
- Theme customization
- More component variants
- Animation library integration
- Design system documentation site
