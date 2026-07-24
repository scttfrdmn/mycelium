// The ordered list of tool surfaces the portal shows. This is the ONLY file that
// changes when a tool joins the portal: add its dep + one surfaces/<tool>.ts, then
// append it here. The shell renders whatever is in this array.
//
// Slice 1 ships instances (spawn) only. truffle (browse, no-auth) and terminal
// (SSM) land in slice 2 as additional entries — the shell already supports them.
import type { ToolSurface } from "./types.js";
import { instancesSurface } from "./instances.js";

export const surfaces: ToolSurface[] = [instancesSurface];

export function findSurface(id: string): ToolSurface | undefined {
  return surfaces.find((s) => s.id === id);
}
