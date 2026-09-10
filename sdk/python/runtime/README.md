# mewisme-meta-runtime

Platform runtime package for `mewisme-meta`. The published project uses one distribution name with platform-specific wheels; each wheel contains the matching `meta-runtime` executable and the `mewisme_meta_runtime.runtime_path()` locator.

Applications normally depend on `mewisme-meta`, which pins this package to the same release version.
