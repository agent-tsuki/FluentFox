import os

def create_modular_react_structure(project_name="fluentfox"):
    """
    Creates a modular React project structure with organized folders
    """
    
    # Define the complete file structure
    structure = {
        "public": {
            "favicon.ico": "",
            "robots.txt": "User-agent: *\nDisallow:"
        },
        "src": {
            "assets": {
                "images": {},
                "icons": {},
                "fonts": {}
            },
            "components": {
                "common": {
                    "Button": {
                        "Button.tsx": "// Common Button component",
                        "Button.module.css": "/* Button styles */",
                        "index.ts": "export { default } from './Button';"
                    },
                    "Modal": {
                        "Modal.tsx": "// Common Modal component",
                        "Modal.module.css": "/* Modal styles */",
                        "index.ts": "export { default } from './Modal';"
                    }
                },
                "layout": {
                    "Header": {
                        "Header.tsx": "// Header component",
                        "Header.module.css": "/* Header styles */",
                        "index.ts": "export { default } from './Header';"
                    },
                    "Footer": {
                        "Footer.tsx": "// Footer component", 
                        "Footer.module.css": "/* Footer styles */",
                        "index.ts": "export { default } from './Footer';"
                    },
                    "Sidebar": {
                        "Sidebar.tsx": "// Sidebar component",
                        "Sidebar.module.css": "/* Sidebar styles */", 
                        "index.ts": "export { default } from './Sidebar';"
                    }
                },
                "ui": {}
            },
            "hooks": {
                "useAuth.ts": "// Custom authentication hook",
                "useLocalStorage.ts": "// Local storage hook", 
                "index.ts": "// Export all hooks"
            },
            "pages": {
                "Home": {
                    "Home.tsx": "// Home page component",
                    "Home.module.css": "/* Home page styles */",
                    "index.ts": "export { default } from './Home';"
                },
                "About": {
                    "About.tsx": "// About page component",
                    "About.module.css": "/* About page styles */", 
                    "index.ts": "export { default } from './About';"
                }
            },
            "services": {
                "api": {
                    "client.ts": "// API client configuration",
                    "endpoints.ts": "// API endpoints",
                    "types.ts": "// API response types"
                },
                "auth": {
                    "authService.ts": "// Authentication service",
                    "tokenManager.ts": "// Token management"
                }
            },
            "styles": {
                "globals.css": "/* Global styles */",
                "variables.css": "/* CSS custom properties */",
                "reset.css": "/* CSS reset styles */"
            },
            "utils": {
                "constants.ts": "// App constants",
                "helpers.ts": "// Utility helper functions",
                "types.ts": "// TypeScript type definitions",
                "validation.ts": "// Form validation utilities"
            },
            "App.tsx": '''import React from 'react';
import './styles/globals.css';

function App() {
  return (
    <div className="App">
      <h1>FluentFox React App</h1>
    </div>
  );
}

export default App;''',
            "main.tsx": '''import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);'''
        }
    }
    
    def create_structure(base_path, structure):
        """Recursively create directories and files"""
        if not os.path.exists(base_path):
            os.makedirs(base_path)
            
        for name, content in structure.items():
            path = os.path.join(base_path, name)
            
            if isinstance(content, dict):
                # It's a directory
                if not os.path.exists(path):
                    os.makedirs(path)
                create_structure(path, content)
            else:
                # It's a file
                with open(path, 'w', encoding='utf-8') as f:
                    f.write(content)
    
    # Create the structure
    base_dir = f'./{project_name}'
    create_structure(base_dir, structure)
    
    print(f"✅ Modular React structure created successfully in '{project_name}/' directory!")
    print("\n📁 Created directories:")
    print("   • src/components/common - Reusable UI components")
    print("   • src/components/layout - Layout components") 
    print("   • src/pages - Page-level components")
    print("   • src/hooks - Custom React hooks")
    print("   • src/services - API and external services")
    print("   • src/utils - Utility functions")
    print("   • src/styles - Global styles and themes")
    print("   • src/assets - Static assets")

# Run the script
if __name__ == "__main__":
    create_modular_react_structure("fluentfox")
