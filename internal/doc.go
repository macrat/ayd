// internal is internal packages for Ayd.
//
// Internal packages do not dependents on each other.
// Dependencies to other package are implemented as a interface like probe.Reporter.
//
// The ayderr package and the testutil package is exception cases for this rule.
// These packages used by other packages.
package internal
