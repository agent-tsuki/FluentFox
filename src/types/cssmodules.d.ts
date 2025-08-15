// CSS Module type declarations
// This lets TypeScript understand imports like `import styles from './file.module.css'`
declare module '*.module.css' {
  const classes: { readonly [key: string]: string };
  export default classes;
}

// Allow importing global/non-module CSS files
declare module '*.css' {
  const css: string;
  export default css;
}
