declare module 'novnc-next' {
  interface RFBOptions {
    credentials?: { password?: string; username?: string } | { [key: string]: string };
    wsProtocols?: string[];
    repeaterID?: string;
    shared?: boolean;
    localCursor?: boolean;
  }
  class RFB {
    constructor(target: HTMLCanvasElement, url: string, options?: RFBOptions);
    addEventListener(event: string, handler: (event: { detail: unknown }) => void): void;
    removeEventListener(event: string, handler: (event: { detail: unknown }) => void): void;
    sendCtrlAltDel(): void;
    sendKey(keysym: number, down?: boolean): void;
    sendCredentials(creds: { username: string; password: string }): void;
    disconnect(): void;
    focus(): void;
    onPasswordRequired?: (callback: () => void) => void;
    onCredentialsRequired?: (callback: () => void) => void;
    onDesktopName?: (callback: (event: { detail: { name: string } }) => void) => void;
    viewOnly?: boolean;
    scaleViewport?: boolean;
    clipViewport?: boolean;
    showDotCursor?: boolean;
  }
  export default RFB;
}