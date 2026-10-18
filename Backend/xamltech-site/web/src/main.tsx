import { hydrateRoot } from "react-dom/client";
import { App } from "./app";
import { bootFromWindow } from "./i18n";
import "./styles/site.css";

const root = document.getElementById("root");
if (root) {
  hydrateRoot(root, <App {...bootFromWindow()} />);
}