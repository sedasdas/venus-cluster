macro_rules! call_rpc {
    ($client:expr, $method:ident, $($arg:expr,)*) => {
        crate::block_on($client.$method($($arg,)*)).map_err(|e| anyhow::anyhow!("rpc error: {:?}", e)).temp()
    };
}

pub(super) use call_rpc;

macro_rules! field_required {
    ($name:ident, $ex:expr) => {
        let $name = $ex
            .with_context(|| format!("{} is required", stringify!(name)))
            .abort()?;
    };
}

pub(super) use field_required;

macro_rules! cloned_required {
    ($name:ident, $ex:expr) => {
        let $name = $ex
            .as_ref()
            .cloned()
            .with_context(|| format!("{} is required", stringify!(name)))
            .abort()?;
    };
}

pub(super) use cloned_required;
