pub fn data_from_a() -> &'static [u8] {
    crate_a::get_data()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn can_access_a() {
        assert!(!data_from_a().is_empty());
    }
}
