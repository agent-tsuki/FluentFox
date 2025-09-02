// Example usage of the new FluentFox UI components
import React, { useState } from 'react';
import { Button, Card, CardHeader, CardBody, CardFooter, CardTitle, CardDescription, Modal } from './src/components/common';

const ComponentShowcase: React.FC = () => {
  const [isModalOpen, setIsModalOpen] = useState(false);

  return (
    <div className="container py-8">
      <h1 className="text-4xl font-bold text-center mb-8 gradient-text">
        FluentFox UI Components
      </h1>
      
      {/* Button Examples */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-6">Buttons</h2>
        <div className="flex flex-wrap gap-4 mb-6">
          <Button variant="primary">Primary Button</Button>
          <Button variant="secondary">Secondary</Button>
          <Button variant="outline">Outline</Button>
          <Button variant="ghost">Ghost</Button>
        </div>
        
        <div className="flex flex-wrap gap-4 mb-6">
          <Button variant="kanji" leftIcon="🀄">Learn Kanji</Button>
          <Button variant="hiragana" leftIcon="🔤">Hiragana</Button>
          <Button variant="katakana" leftIcon="🔤">Katakana</Button>
          <Button variant="grammar" leftIcon="📝">Grammar</Button>
        </div>
        
        <div className="flex flex-wrap gap-4 items-center">
          <Button size="xs">Extra Small</Button>
          <Button size="sm">Small</Button>
          <Button size="md">Medium</Button>
          <Button size="lg">Large</Button>
          <Button size="xl">Extra Large</Button>
        </div>
      </section>

      {/* Card Examples */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-6">Cards</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          <Card variant="default" clickable>
            <CardHeader>
              <CardTitle>Default Card</CardTitle>
            </CardHeader>
            <CardBody>
              <CardDescription>
                This is a default card with hover effects and clean styling.
              </CardDescription>
            </CardBody>
            <CardFooter>
              <Button size="sm">Learn More</Button>
            </CardFooter>
          </Card>

          <Card variant="glass">
            <CardHeader>
              <CardTitle>Glass Card</CardTitle>
            </CardHeader>
            <CardBody>
              <CardDescription>
                Beautiful glassmorphism effect with backdrop blur.
              </CardDescription>
            </CardBody>
          </Card>

          <Card variant="gradient">
            <CardHeader>
              <CardTitle>Gradient Card</CardTitle>
            </CardHeader>
            <CardBody>
              <CardDescription>
                Eye-catching gradient background with white text.
              </CardDescription>
            </CardBody>
          </Card>
        </div>
      </section>

      {/* Modal Example */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-6">Modal</h2>
        <Button onClick={() => setIsModalOpen(true)}>
          Open Modal
        </Button>
        
        <Modal
          isOpen={isModalOpen}
          onClose={() => setIsModalOpen(false)}
          title="Example Modal"
          size="md"
        >
          <p>This is a fully-featured modal with:</p>
          <ul className="list-disc list-inside mt-4 space-y-2">
            <li>Portal-based rendering</li>
            <li>Focus management</li>
            <li>Keyboard navigation (ESC to close)</li>
            <li>Backdrop click to close</li>
            <li>Mobile responsive design</li>
            <li>Accessibility features</li>
          </ul>
        </Modal>
      </section>

      {/* Utility Classes Demo */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-6">Utility Classes</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="p-6 bg-white rounded-xl shadow-md">
            <h3 className="font-semibold mb-3">Spacing</h3>
            <div className="space-y-2">
              <div className="p-2 bg-primary-100">p-2</div>
              <div className="p-4 bg-primary-200">p-4</div>
              <div className="p-6 bg-primary-300">p-6</div>
            </div>
          </div>
          
          <div className="p-6 bg-white rounded-xl shadow-md">
            <h3 className="font-semibold mb-3">Colors</h3>
            <div className="space-y-2">
              <div className="w-full h-8 bg-primary rounded"></div>
              <div className="w-full h-8 bg-secondary rounded"></div>
              <div className="w-full h-8 bg-success rounded"></div>
            </div>
          </div>
        </div>
      </section>
    </div>
  );
};

export default ComponentShowcase;
