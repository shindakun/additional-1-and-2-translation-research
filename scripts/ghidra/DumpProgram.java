// Writes decompiled C for every function, a function list, and a string
// cross-reference table to the directory given as the first script argument.
// @category KFA

import java.io.File;
import java.io.PrintWriter;
import java.nio.charset.StandardCharsets;

import ghidra.app.decompiler.DecompInterface;
import ghidra.app.decompiler.DecompileResults;
import ghidra.app.script.GhidraScript;
import ghidra.program.model.listing.Data;
import ghidra.program.model.listing.DataIterator;
import ghidra.program.model.listing.Function;
import ghidra.program.model.symbol.Reference;

public class DumpProgram extends GhidraScript {
	@Override
	protected void run() throws Exception {
		File out = new File(getScriptArgs()[0]);
		out.mkdirs();

		DecompInterface dec = new DecompInterface();
		dec.openProgram(currentProgram);
		try (PrintWriter c = new PrintWriter(new File(out, "decomp.c"), StandardCharsets.UTF_8);
				PrintWriter f = new PrintWriter(new File(out, "functions.tsv"), StandardCharsets.UTF_8)) {
			f.println("entry\tname\tsize\tcallers");
			for (Function fn : currentProgram.getFunctionManager().getFunctions(true)) {
				if (monitor.isCancelled()) {
					break;
				}
				f.printf("%s\t%s\t%d\t%d%n", fn.getEntryPoint(), fn.getName(),
					fn.getBody().getNumAddresses(), fn.getCallingFunctions(monitor).size());
				DecompileResults r = dec.decompileFunction(fn, 60, monitor);
				c.printf("// ==== %s %s%n", fn.getEntryPoint(), fn.getName());
				if (r.decompileCompleted()) {
					c.println(r.getDecompiledFunction().getC());
				}
				else {
					c.println("// decompile failed: " + r.getErrorMessage());
				}
			}
		}
		dec.dispose();

		try (PrintWriter s = new PrintWriter(new File(out, "strings.tsv"), StandardCharsets.UTF_8)) {
			s.println("address\tfunctions\tvalue");
			DataIterator it = currentProgram.getListing().getDefinedData(true);
			while (it.hasNext()) {
				Data d = it.next();
				if (!d.hasStringValue()) {
					continue;
				}
				StringBuilder users = new StringBuilder();
				for (Reference ref : getReferencesTo(d.getAddress())) {
					Function fn = getFunctionContaining(ref.getFromAddress());
					users.append(fn == null ? ref.getFromAddress().toString() : fn.getName()).append(',');
				}
				String v = String.valueOf(d.getValue()).replace("\t", "\\t").replace("\n", "\\n");
				s.printf("%s\t%s\t%s%n", d.getAddress(), users, v);
			}
		}
		println("wrote " + out);
	}
}
